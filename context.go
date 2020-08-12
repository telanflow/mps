package mps

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"time"
)

// Http method not support
var MethodNotSupportErr = errors.New("request method not support")

// Context for the request
// which contains Middleware, Transport, and other values
type Context struct {
	// context.Context
	Context context.Context

	// Request context-dependent requests
	Request *http.Request

	// Response is associated with Request
	Response *http.Response

	// Transport is used for global HTTP requests, and it will be reused.
	Transport *http.Transport

	// In some cases it is not always necessary to remove the proxy headers.
	// For example, cascade proxy
	KeepProxyHeaders bool

	// In some cases it is not always necessary to reset the headers.
	KeepClientHeaders bool

	// KeepDestinationHeaders indicates the proxy should retain any headers
	// present in the http.Response before proxying
	KeepDestinationHeaders bool

	// middlewares ACTS on Request and Response.
	// It's going to be reused by the Context
	// mi is the index subscript of the middlewares traversal
	// the default value for the index is -1
	mi          int
	middlewares []Middleware
}

// Create a Context
func NewContext() *Context {
	return &Context{
		Context: context.Background(),
		// Cannot reuse one Transport because multiple proxy can collide with each other
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   15 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			Proxy:                 http.ProxyFromEnvironment,
		},
		Request:                nil,
		Response:               nil,
		KeepProxyHeaders:       false,
		KeepClientHeaders:      false,
		KeepDestinationHeaders: false,
		mi:                     -1,
		middlewares:            make([]Middleware, 0),
	}
}

// Use registers an Middleware to proxy
func (ctx *Context) Use(middleware ...Middleware) {
	if ctx.middlewares == nil {
		ctx.middlewares = make([]Middleware, 0)
	}
	ctx.middlewares = append(ctx.middlewares, middleware...)
}

// UseFunc registers an MiddlewareFunc to proxy
func (ctx *Context) UseFunc(fns ...MiddlewareFunc) {
	if ctx.middlewares == nil {
		ctx.middlewares = make([]Middleware, 0)
	}
	for _, fn := range fns {
		ctx.middlewares = append(ctx.middlewares, fn)
	}
}

// Next to exec middlewares
// Execute the next middleware as a linked list. "ctx.Next(req)"
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
func (ctx *Context) Next(req *http.Request) (*http.Response, error) {
	var (
		total = len(ctx.middlewares)
		err   error
	)
	ctx.mi++
	if ctx.mi >= total {
		ctx.mi = -1
		// To make the middleware available to the tunnel proxy,
		// no response is obtained when the request method is equal to Connect
		if req.Method == http.MethodConnect {
			return nil, MethodNotSupportErr
		}
		return ctx.RoundTrip(req)
	}

	middleware := ctx.middlewares[ctx.mi]
	ctx.Response, err = middleware.Handle(req, ctx)
	ctx.mi = -1
	return ctx.Response, err
}

// RoundTrip implements the RoundTripper interface.
//
// For higher-level HTTP client support (such as handling of cookies
// and redirects), see Get, Post, and the Client type.
//
// Like the RoundTripper interface, the error types returned
// by RoundTrip are unspecified.
func (ctx *Context) RoundTrip(req *http.Request) (*http.Response, error) {
	// These Headers must be reset when a client Request is issued to reuse a Request
	if !ctx.KeepClientHeaders {
		ResetClientHeaders(req)
	}

	// In some cases it is not always necessary to remove the Proxy Header.
	// For example, cascade proxy
	if !ctx.KeepProxyHeaders {
		RemoveProxyHeaders(req)
	}

	if ctx.Transport != nil {
		return ctx.Transport.RoundTrip(req)
	}
	return DefaultTransport.RoundTrip(req)
}

// WithRequest get the Context of the request
func (ctx *Context) WithRequest(req *http.Request) *Context {
	return &Context{
		Context:                context.Background(),
		Request:                req,
		Response:               nil,
		KeepProxyHeaders:       ctx.KeepProxyHeaders,
		KeepClientHeaders:      ctx.KeepClientHeaders,
		KeepDestinationHeaders: ctx.KeepDestinationHeaders,
		Transport:              ctx.Transport,
		mi:                     -1,
		middlewares:            ctx.middlewares,
	}
}

// ResetClientHeaders These Headers must be reset when a client Request is issued to reuse a Request
func ResetClientHeaders(r *http.Request) {
	// this must be reset when serving a request with the client
	r.RequestURI = ""
	// If no Accept-Encoding header exists, Transport will add the headers it can accept
	// and would wrap the response body with the relevant reader.
	r.Header.Del("Accept-Encoding")
}

// Hop-by-hop headers. These are removed when sent to the backend.
// As of RFC 7230, hop-by-hop headers are required to appear in the
// Connection header field. These are the headers defined by the
// obsoleted RFC 2616 (section 13.5.1) and are used for backward
// compatibility.
func RemoveProxyHeaders(r *http.Request) {
	// RFC 2616 (section 13.5.1)
	// https://www.ietf.org/rfc/rfc2616.txt
	r.Header.Del("Proxy-Connection")
	r.Header.Del("Proxy-Authenticate")
	r.Header.Del("Proxy-Authorization")
	// Connection, Authenticate and Authorization are single hop Header:
	// http://www.w3.org/Protocols/rfc2616/rfc2616.txt
	// 14.10 Connection
	//   The Connection general-header field allows the sender to specify
	//   options that are desired for that particular connection and MUST NOT
	//   be communicated by proxies over further connections.

	// When server reads http request it sets req.Close to true if
	// "Connection" header contains "close".
	// https://github.com/golang/go/blob/master/src/net/http/request.go#L1080
	// Later, transfer.go adds "Connection: close" back when req.Close is true
	// https://github.com/golang/go/blob/master/src/net/http/transfer.go#L275
	// That's why tests that checks "Connection: close" removal fail
	if r.Header.Get("Connection") == "close" {
		r.Close = false
	}
	r.Header.Del("Connection")
}
