package mps

import "net/http"

// Middleware will "tamper" with the request coming to the proxy server
type Middleware interface {
	// Handle execute the next middleware as a linked list. "ctx.Next(req)"
	// eg:
	// 		func Handle(req *http.Request, ctx *Context) (*http.Response, error) {
	//				// You can do anything to modify the http.Request ...
	// 				resp, err := ctx.Next(req)
	// 				// You can do anything to modify the http.Response ...
	//				return resp, err
	// 		}
	//
	// Alternatively, you can simply return the response without executing `ctx.Next()`,
	// which will interrupt subsequent middleware execution.
	Handle(req *http.Request, ctx *Context) (*http.Response, error)
}

// MiddlewareFunc A wrapper that would convert a function to a Middleware interface type
type MiddlewareFunc func(req *http.Request, ctx *Context) (*http.Response, error)

// Handle Middleware.Handle(req, ctx) <=> MiddlewareFunc(req, ctx)
func (f MiddlewareFunc) Handle(req *http.Request, ctx *Context) (*http.Response, error) {
	return f(req, ctx)
}
