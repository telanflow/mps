package mps

import "net/http"

type RequestHandle interface {
	Handle(req *http.Request, ctx *Context) (*http.Request, *http.Response)
}

// A wrapper that would convert a function to a RequestHandle interface type
type RequestHandleFunc func(req *http.Request, ctx *Context) (*http.Request, *http.Response)

// RequestHandle.Handle(req, ctx) <=> RequestHandleFunc(req, ctx)
func (f RequestHandleFunc) Handle(req *http.Request, ctx *Context) (*http.Request, *http.Response) {
	return f(req, ctx)
}

type ResponseHandle interface {
	Handle(resp *http.Response, err error, ctx *Context) (*http.Response, error)
}

// A wrapper that would convert a function to a ResponseHandle interface type
type ResponseHandleFunc func(resp *http.Response, err error, ctx *Context) (*http.Response, error)

// ResponseHandle.Handle(resp, ctx) <=> ResponseHandleFunc(resp, ctx)
func (f ResponseHandleFunc) Handle(resp *http.Response, err error, ctx *Context) (*http.Response, error) {
	return f(resp, err, ctx)
}
