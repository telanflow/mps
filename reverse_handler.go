package mps

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"

	"github.com/telanflow/mps/pool"
)

// ReverseHandler is a reverse proxy server implementation
type ReverseHandler struct {
	Ctx        *Context
	BufferPool httputil.BufferPool
}

// NewReverseHandler Create a reverse proxy
func NewReverseHandler() *ReverseHandler {
	return &ReverseHandler{
		Ctx:        NewContext(),
		BufferPool: pool.DefaultBuffer,
	}
}

// Standard net/http function. You can use it alone
func (reverse *ReverseHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Copying a Context preserves the Transport, Middleware
	ctx := reverse.Ctx.WithRequest(req)
	resp, err := ctx.Next(req)
	if err != nil {
		http.Error(rw, err.Error(), 502)
		return
	}
	defer resp.Body.Close()

	var (
		// Body buffer
		buffer = new(bytes.Buffer)
		// Body size
		bufferSize int64
	)

	buf := reverse.buffer().Get()
	bufferSize, err = io.CopyBuffer(buffer, resp.Body, buf)
	reverse.buffer().Put(buf)
	if err != nil {
		http.Error(rw, err.Error(), 502)
		return
	}

	resp.ContentLength = bufferSize
	resp.Header.Set("Content-Length", strconv.Itoa(int(bufferSize)))
	copyHeaders(rw.Header(), resp.Header, reverse.Ctx.KeepDestinationHeaders)
	rw.WriteHeader(resp.StatusCode)
	_, err = buffer.WriteTo(rw)
}

// Use registers an Middleware to proxy
func (reverse *ReverseHandler) Use(middleware ...Middleware) {
	reverse.Ctx.Use(middleware...)
}

// UseFunc registers an MiddlewareFunc to proxy
func (reverse *ReverseHandler) UseFunc(fus ...MiddlewareFunc) {
	reverse.Ctx.UseFunc(fus...)
}

// OnRequest filter requests through Filters
func (reverse *ReverseHandler) OnRequest(filters ...Filter) *ReqFilterGroup {
	return &ReqFilterGroup{ctx: reverse.Ctx, filters: filters}
}

// OnResponse filter response through Filters
func (reverse *ReverseHandler) OnResponse(filters ...Filter) *RespFilterGroup {
	return &RespFilterGroup{ctx: reverse.Ctx, filters: filters}
}

// Get buffer pool
func (reverse *ReverseHandler) buffer() httputil.BufferPool {
	if reverse.BufferPool != nil {
		return reverse.BufferPool
	}
	return pool.DefaultBuffer
}

// Transport
func (reverse *ReverseHandler) Transport() *http.Transport {
	return reverse.Ctx.Transport
}
