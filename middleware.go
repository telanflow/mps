package mps

import "net/http"

type Middleware interface {
	Handle(req *http.Request, ctx *Context) (*http.Response, error)
}

type MiddlewareFunc func(req *http.Request, ctx *Context) (*http.Response, error)

func (f MiddlewareFunc) Handle(req *http.Request, ctx *Context) (*http.Response, error) {
	return f(req, ctx)
}
