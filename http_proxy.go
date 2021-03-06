package mps

import (
	"errors"
	"fmt"
	"net"
	"net/http"
)

// The basic proxy type. Implements http.Handler.
type HttpProxy struct {
	// Handles Connect requests use the TunnelHandler by default
	HandleConnect http.Handler

	// HTTP requests use the ForwardHandler by default
	HttpHandler http.Handler

	// HTTP requests use the ReverseHandler by default
	ReverseHandler http.Handler

	// Client request Context
	Ctx *Context
}

func NewHttpProxy() *HttpProxy {
	// default Context with Proxy
	ctx := NewContext()
	return &HttpProxy{
		Ctx: ctx,
		// default handles Connect method
		HandleConnect: &TunnelHandler{Ctx: ctx},
		// default handles HTTP request
		HttpHandler: &ForwardHandler{Ctx: ctx},
		// default Reverse proxy
		ReverseHandler: &ReverseHandler{Ctx: ctx},
	}
}

// Standard net/http function.
func (proxy *HttpProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodConnect {
		proxy.HandleConnect.ServeHTTP(rw, req)
		return
	}

	// reverse proxy http request for example:
	//		GET / HTTP/1.1
	//		Host: www.example.com
	//		Connection: keep-alive
	//
	// forward proxy http request for example :
	// 		GET http://www.example.com/ HTTP/1.1
	// 		Host: www.example.com
	// 		Proxy-Connection: keep-alive
	//
	// Determines whether the path is absolute
	if !req.URL.IsAbs() {
		proxy.ReverseHandler.ServeHTTP(rw, req)
	} else {
		proxy.HttpHandler.ServeHTTP(rw, req)
	}
}

// Use registers an Middleware to proxy
func (proxy *HttpProxy) Use(middleware ...Middleware) {
	proxy.Ctx.Use(middleware...)
}

// UseFunc registers an MiddlewareFunc to proxy
func (proxy *HttpProxy) UseFunc(fus ...MiddlewareFunc) {
	proxy.Ctx.UseFunc(fus...)
}

// OnRequest filter requests through Filters
func (proxy *HttpProxy) OnRequest(filters ...Filter) *ReqFilterGroup {
	return &ReqFilterGroup{ctx: proxy.Ctx, filters: filters}
}

// OnResponse filter response through Filters
func (proxy *HttpProxy) OnResponse(filters ...Filter) *RespFilterGroup {
	return &RespFilterGroup{ctx: proxy.Ctx, filters: filters}
}

// Transport get http.Transport instance
func (proxy *HttpProxy) Transport() *http.Transport {
	return proxy.Ctx.Transport
}

// hijacker an HTTP handler to take over the connection.
func hijacker(rw http.ResponseWriter) (conn net.Conn, err error) {
	hij, ok := rw.(http.Hijacker)
	if !ok {
		err = errors.New("not a hijacker")
		return
	}

	conn, _, err = hij.Hijack()
	if err != nil {
		err = fmt.Errorf("cannot hijack connection %v", err)
	}
	return
}

func copyHeaders(dst, src http.Header, keepDestHeaders bool) {
	if !keepDestHeaders {
		for k := range dst {
			dst.Del(k)
		}
	}
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}
