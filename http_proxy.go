package mps

import (
	"errors"
	"fmt"
	"net"
	"net/http"
)

type HttpProxy struct {
	// HTTPS requests use the TunnelHandler proxy by default
	HttpsHandler http.Handler

	// HTTP requests use the ForwardHandler proxy by default
	HttpHandler http.Handler

	Ctx *Context
}

func NewHttpProxy() *HttpProxy {
	// default Context with Proxy
	ctx := NewContext()

	return &HttpProxy{
		Ctx:    ctx,
		// default HTTP proxy
		HttpHandler:  &ForwardHandler{Ctx: ctx},
		// default HTTPS proxy
		HttpsHandler: &TunnelHandler{Ctx: ctx},
	}
}

func (proxy *HttpProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodConnect {
		proxy.HttpsHandler.ServeHTTP(rw, req)
	}
	proxy.HttpHandler.ServeHTTP(rw, req)
}

func (proxy *HttpProxy) Use(middleware ...Middleware) {
	proxy.Ctx.Use(middleware...)
}

func (proxy *HttpProxy) UseFunc(fus ...MiddlewareFunc) {
	proxy.Ctx.UseFunc(fus...)
}

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
	// curl can add that, see
	// https://jdebp.eu./FGA/web-proxy-connection-header.html

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
