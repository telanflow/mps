package mps

import (
	"errors"
	"fmt"
	"net"
	"net/http"
)

// The basic proxy type. Implements http.Handler.
type HttpProxy struct {
	// HTTPS requests use the TunnelHandler proxy by default
	HttpsHandler http.Handler

	// HTTP requests use the ForwardHandler proxy by default
	HttpHandler http.Handler

	// HTTP requests use the ReverseHandler proxy by default
	ReverseHandler http.Handler

	Ctx *Context
}

func NewHttpProxy() *HttpProxy {
	// default Context with Proxy
	ctx := NewContext()

	return &HttpProxy{
		Ctx: ctx,
		// default HTTP proxy
		HttpHandler: &ForwardHandler{Ctx: ctx},
		// default HTTPS proxy
		HttpsHandler: &TunnelHandler{Ctx: ctx},
		// default Reverse proxy
		ReverseHandler: &ReverseHandler{Ctx: ctx},
	}
}

// Standard net/http function.
func (proxy *HttpProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodConnect {
		proxy.HttpsHandler.ServeHTTP(rw, req)
	}

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
func (proxy *HttpProxy) OnRequest(filters ...Filter) *ReqCondition {
	return &ReqCondition{ctx: proxy.Ctx, filters: filters}
}

// OnResponse filter response through Filters
func (proxy *HttpProxy) OnResponse(filters ...Filter) *RespCondition {
	return &RespCondition{ctx: proxy.Ctx, filters: filters}
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

// Hop-by-hop headers. These are removed when sent to the backend.
// As of RFC 7230, hop-by-hop headers are required to appear in the
// Connection header field. These are the headers defined by the
// obsoleted RFC 2616 (section 13.5.1) and are used for backward
// compatibility.
func removeProxyHeaders(r *http.Request) {
	r.RequestURI = "" // this must be reset when serving a request with the client
	// If no Accept-Encoding header exists, Transport will add the headers it can accept
	// and would wrap the response body with the relevant reader.
	r.Header.Del("Accept-Encoding")
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
