package mps

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"time"

	"github.com/telanflow/mps/pool"
)

var (
	HttpTunnelOk   = []byte("HTTP/1.0 200 Connection Established\r\n\r\n")
	HttpTunnelFail = []byte("HTTP/1.1 502 Bad Gateway\r\n\r\n")
	hasPort        = regexp.MustCompile(`:\d+$`)
)

// TunnelHandler The tunnel proxy type. Implements http.Handler.
type TunnelHandler struct {
	Ctx           *Context
	BufferPool    httputil.BufferPool
	ConnContainer pool.ConnContainer
}

// NewTunnelHandler Create a tunnel handler
func NewTunnelHandler() *TunnelHandler {
	return &TunnelHandler{
		Ctx:        NewContext(),
		BufferPool: pool.DefaultBuffer,
	}
}

// NewTunnelHandlerWithContext Create a tunnel handler with Context
func NewTunnelHandlerWithContext(ctx *Context) *TunnelHandler {
	return &TunnelHandler{
		Ctx:        ctx,
		BufferPool: pool.DefaultBuffer,
	}
}

// Standard net/http function. You can use it alone
func (tunnel *TunnelHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// execution middleware
	ctx := tunnel.Ctx.WithRequest(req)
	resp, err := ctx.Next(req)
	if err != nil && err != MethodNotSupportErr {
		if resp != nil {
			copyHeaders(rw.Header(), resp.Header, tunnel.Ctx.KeepDestinationHeaders)
			rw.WriteHeader(resp.StatusCode)
			buf := tunnel.buffer().Get()
			_, err = io.CopyBuffer(rw, resp.Body, buf)
			tunnel.buffer().Put(buf)
		}
		return
	}

	// hijacker connection
	proxyClient, err := hijacker(rw)
	if err != nil {
		http.Error(rw, err.Error(), 502)
		return
	}

	var (
		u              *url.URL = nil
		targetConn     net.Conn = nil
		targetAddr              = hostAndPort(req.URL.Host)
		isCascadeProxy          = false
	)
	if tunnel.Ctx.Transport != nil && tunnel.Ctx.Transport.Proxy != nil {
		u, err = tunnel.Ctx.Transport.Proxy(req)
		if err != nil {
			ConnError(proxyClient)
			return
		}
		if u != nil {
			// connect addr eg. "localhost:80"
			targetAddr = hostAndPort(u.Host)
			isCascadeProxy = true
		}
	}

	// connect to targetAddr
	targetConn, err = tunnel.connContainer().Get(targetAddr)
	if err != nil {
		targetConn, err = tunnel.ConnectDial("tcp", targetAddr)
		if err != nil {
			ConnError(proxyClient)
			return
		}
	}

	// If the ConnContainer is exists,
	// When io.CopyBuffer is complete,
	// put the idle connection into the ConnContainer so can reuse it next time
	defer func() {
		err := tunnel.connContainer().Put(targetConn)
		if err != nil {
			// put conn fail, conn must be closed
			_ = targetConn.Close()
		}
	}()

	// The cascade proxy needs to forward the request
	if isCascadeProxy {
		// The cascade proxy needs to send it as-is
		_ = req.Write(targetConn)
	} else {
		// Tell client that the tunnel is ready
		_, _ = proxyClient.Write(HttpTunnelOk)
	}

	go func() {
		buf := tunnel.buffer().Get()
		_, _ = io.CopyBuffer(targetConn, proxyClient, buf)
		tunnel.buffer().Put(buf)
		_ = proxyClient.Close()
	}()
	buf := tunnel.buffer().Get()
	_, _ = io.CopyBuffer(proxyClient, targetConn, buf)
	tunnel.buffer().Put(buf)
}

// Use registers an Middleware to proxy
func (tunnel *TunnelHandler) Use(middleware ...Middleware) {
	tunnel.Ctx.Use(middleware...)
}

// UseFunc registers an MiddlewareFunc to proxy
func (tunnel *TunnelHandler) UseFunc(fus ...MiddlewareFunc) {
	tunnel.Ctx.UseFunc(fus...)
}

// OnRequest filter requests through Filters
func (tunnel *TunnelHandler) OnRequest(filters ...Filter) *ReqFilterGroup {
	return &ReqFilterGroup{ctx: tunnel.Ctx, filters: filters}
}

// OnResponse filter response through Filters
func (tunnel *TunnelHandler) OnResponse(filters ...Filter) *RespFilterGroup {
	return &RespFilterGroup{ctx: tunnel.Ctx, filters: filters}
}

func (tunnel *TunnelHandler) ConnectDial(network, addr string) (net.Conn, error) {
	if tunnel.Ctx.Transport != nil && tunnel.Ctx.Transport.DialContext != nil {
		return tunnel.Ctx.Transport.DialContext(tunnel.context(), network, addr)
	}
	return net.DialTimeout(network, addr, 30*time.Second)
}

// Transport get http.Transport instance
func (tunnel *TunnelHandler) Transport() *http.Transport {
	return tunnel.Ctx.Transport
}

// get a context.Context
func (tunnel *TunnelHandler) context() context.Context {
	if tunnel.Ctx.Context != nil {
		return tunnel.Ctx.Context
	}
	return context.Background()
}

// Get buffer pool
func (tunnel *TunnelHandler) buffer() httputil.BufferPool {
	if tunnel.BufferPool != nil {
		return tunnel.BufferPool
	}
	return pool.DefaultBuffer
}

// Get a conn pool
func (tunnel *TunnelHandler) connContainer() pool.ConnContainer {
	if tunnel.ConnContainer != nil {
		return tunnel.ConnContainer
	}
	return pool.DefaultConnProvider
}

func hostAndPort(addr string) string {
	if !hasPort.MatchString(addr) {
		addr += ":80"
	}
	return addr
}

func ConnError(w net.Conn) {
	_, _ = w.Write(HttpTunnelFail)
	_ = w.Close()
}
