package mps

import (
	"context"
	"crypto/tls"
	"net/http"
)

type Context struct {
	Context context.Context
	Request   *http.Request
	Response  *http.Response
	Transport *http.Transport

	// In some cases it is not always necessary to remove the Proxy Header.
	// For example, cascade proxy
	KeepHeader bool

	// KeepDestinationHeaders indicates the proxy should retain any headers
	// present in the http.Response before proxying
	KeepDestinationHeaders bool

	// requests Middleware
	mi          int
	middlewares []Middleware
}

func NewContext() *Context {
	return &Context{
		Context: context.Background(),
		Transport: &http.Transport{
			//DialContext: (&net.Dialer{
			//	Timeout:   15 * time.Second,
			//	KeepAlive: 30 * time.Second,
			//	DualStack: true,
			//}).DialContext,
			////ForceAttemptHTTP2:     true,
			//MaxIdleConns:          100,
			//IdleConnTimeout:       90 * time.Second,
			//TLSHandshakeTimeout:   10 * time.Second,
			//ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			Proxy:                 http.ProxyFromEnvironment,
		},
		Request:     nil,
		Response:    nil,
		KeepHeader: false,
		KeepDestinationHeaders: false,
		mi:          -1,
		middlewares: make([]Middleware, 0),
	}
}

func (ctx *Context) Use(middleware ...Middleware) {
	if ctx.middlewares == nil {
		ctx.middlewares = make([]Middleware, 0)
	}

	ctx.middlewares = append(ctx.middlewares, middleware...)
}

func (ctx *Context) UseFunc(fns ...MiddlewareFunc) {
	if ctx.middlewares == nil {
		ctx.middlewares = make([]Middleware, 0)
	}

	for _, fn := range fns {
		ctx.middlewares = append(ctx.middlewares, fn)
	}
}

func (ctx *Context) Next(req *http.Request) (*http.Response, error) {
	var (
		err   error
		total = len(ctx.middlewares)
	)
	ctx.mi++
	if ctx.mi >= total {
		ctx.mi = -1
		return ctx.Transport.RoundTrip(req)
	}

	middleware := ctx.middlewares[ctx.mi]
	ctx.Response, err = middleware.Handle(req, ctx)
	ctx.mi = -1
	return ctx.Response, err
}

func (ctx *Context) Copy() *Context {
	return &Context{
		Context: context.Background(),
		Request:                nil,
		Response:               nil,
		KeepHeader:             false,
		KeepDestinationHeaders: false,
		Transport:              ctx.Transport,
		mi:                     -1,
		middlewares:            ctx.middlewares,
	}
}
