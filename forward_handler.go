package mps

import (
	"bytes"
	"github.com/telanflow/mps/pool"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
)

// The forward proxy type. Implements http.Handler.
type ForwardHandler struct {
	Ctx        *Context
	BufferPool httputil.BufferPool
}

// Create a ForwardHandler
func NewForwardHandler() *ForwardHandler {
	return &ForwardHandler{
		Ctx:        NewContext(),
		BufferPool: pool.DefaultBuffer,
	}
}

// Create a ForwardHandler with Context
func NewForwardHandlerWithContext(ctx *Context) *ForwardHandler {
	return &ForwardHandler{
		Ctx: ctx,
	}
}

// Standard net/http function. You can use it alone
func (forward *ForwardHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Copying a Context preserves the Transport, Middleware
	ctx := forward.Ctx.Copy()
	ctx.Request = req

	// In some cases it is not always necessary to remove the Proxy Header.
	// For example, cascade proxy
	if !forward.Ctx.KeepHeader {
		removeProxyHeaders(req)
	}

	resp, err := ctx.Next(req)
	if err != nil {
		http.Error(rw, err.Error(), 502)
		return
	}

	bodyRes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(rw, err.Error(), 502)
		return
	}
	_ = resp.Body.Close()

	// http.ResponseWriter will take care of filling the correct response length
	// Setting it now, might impose wrong value, contradicting the actual new
	// body the user returned.
	// We keep the original body to remove the header only if things changed.
	// This will prevent problems with HEAD requests where there's no body, yet,
	// the Content-Length header should be set.
	if resp.ContentLength != int64(len(bodyRes)) {
		resp.Header.Del("Content-Length")
	}

	copyHeaders(rw.Header(), resp.Header, forward.Ctx.KeepDestinationHeaders)
	rw.WriteHeader(resp.StatusCode)

	body := ioutil.NopCloser(bytes.NewReader(bodyRes))
	buf := forward.BufferPool.Get()
	_, err = io.CopyBuffer(rw, body, buf)
	forward.BufferPool.Put(buf)
	_ = body.Close()
	if err != nil {
		http.Error(rw, err.Error(), 502)
		return
	}
}

func (forward *ForwardHandler) Transport() *http.Transport {
	return forward.Ctx.Transport
}
