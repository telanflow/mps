package mps

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
)

// ReverseHandler is a reverse proxy server implementation
type ReverseHandler struct {
	Ctx *Context
}

func NewReverseHandler() *ReverseHandler {
	return &ReverseHandler{
		Ctx: NewContext(),
	}
}

// Standard net/http function. You can use it alone
func (reverse *ReverseHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Copying a Context preserves the Transport, Middleware
	ctx := reverse.Ctx.Copy()
	ctx.Request = req

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
	resp.Body.Close()

	// http.ResponseWriter will take care of filling the correct response length
	// Setting it now, might impose wrong value, contradicting the actual new
	// body the user returned.
	// We keep the original body to remove the header only if things changed.
	// This will prevent problems with HEAD requests where there's no body, yet,
	// the Content-Length header should be set.
	if resp.ContentLength != int64(len(bodyRes)) {
		resp.Header.Del("Content-Length")
	}

	copyHeaders(rw.Header(), resp.Header, reverse.Ctx.KeepDestinationHeaders)
	rw.WriteHeader(resp.StatusCode)

	body := ioutil.NopCloser(bytes.NewReader(bodyRes))
	_, err = io.Copy(rw, body)
	body.Close()
	if err != nil {
		http.Error(rw, err.Error(), 502)
		return
	}
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
func (reverse *ReverseHandler) OnRequest(filters ...Filter) *ReqCondition {
	return &ReqCondition{ctx: reverse.Ctx, filters: filters}
}

// OnResponse filter response through Filters
func (reverse *ReverseHandler) OnResponse(filters ...Filter) *RespCondition {
	return &RespCondition{ctx: reverse.Ctx, filters: filters}
}
