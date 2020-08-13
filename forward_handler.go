package mps

import (
	"github.com/telanflow/mps/pool"
	"io"
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
		Ctx:        ctx,
		BufferPool: pool.DefaultBuffer,
	}
}

// Standard net/http function. You can use it alone
func (forward *ForwardHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Copying a Context preserves the Transport, Middleware
	ctx := forward.Ctx.WithRequest(req)
	resp, err := ctx.Next(req)
	if err != nil {
		http.Error(rw, err.Error(), 502)
		return
	}

	copyHeaders(rw.Header(), resp.Header, forward.Ctx.KeepDestinationHeaders)
	rw.WriteHeader(resp.StatusCode)

	buf := forward.buffer().Get()
	_, err = io.CopyBuffer(rw, resp.Body, buf)
	forward.buffer().Put(buf)
	_ = resp.Body.Close()
	if err != nil {
		http.Error(rw, err.Error(), 502)
		return
	}
}

// Use registers an Middleware to proxy
func (forward *ForwardHandler) Use(middleware ...Middleware) {
	forward.Ctx.Use(middleware...)
}

// UseFunc registers an MiddlewareFunc to proxy
func (forward *ForwardHandler) UseFunc(fus ...MiddlewareFunc) {
	forward.Ctx.UseFunc(fus...)
}

// OnRequest filter requests through Filters
func (forward *ForwardHandler) OnRequest(filters ...Filter) *ReqFilterGroup {
	return &ReqFilterGroup{ctx: forward.Ctx, filters: filters}
}

// OnResponse filter response through Filters
func (forward *ForwardHandler) OnResponse(filters ...Filter) *RespFilterGroup {
	return &RespFilterGroup{ctx: forward.Ctx, filters: filters}
}

// Transport
func (forward *ForwardHandler) Transport() *http.Transport {
	return forward.Ctx.Transport
}

// Get buffer pool
func (forward *ForwardHandler) buffer() httputil.BufferPool {
	if forward.BufferPool != nil {
		return forward.BufferPool
	}
	return pool.DefaultBuffer
}
