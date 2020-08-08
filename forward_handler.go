package mps

import (
	"io"
	"net/http"
)

type ForwardHandler struct {
	Ctx *Context
}

func NewForwardHandler() *ForwardHandler {
	return &ForwardHandler{
		Ctx: NewContext(),
	}
}

func NewForwardHandlerWithContext(ctx *Context) *ForwardHandler {
	return &ForwardHandler{
		Ctx: ctx,
	}
}

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
		http.Error(rw, err.Error(), 500)
		return
	}

	origBody := resp.Body
	defer origBody.Close()

	// http.ResponseWriter will take care of filling the correct response length
	// Setting it now, might impose wrong value, contradicting the actual new
	// body the user returned.
	// We keep the original body to remove the header only if things changed.
	// This will prevent problems with HEAD requests where there's no body, yet,
	// the Content-Length header should be set.
	if origBody != resp.Body {
		resp.Header.Del("Content-Length")
	}
	copyHeaders(rw.Header(), resp.Header, forward.Ctx.KeepDestinationHeaders)
	rw.WriteHeader(resp.StatusCode)
	io.Copy(rw, resp.Body)
	resp.Body.Close()
}

func (forward *ForwardHandler) Transport() *http.Transport {
	return forward.Ctx.Transport
}
