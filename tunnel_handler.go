package mps

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
)

var (
	HttpTunnelOk   = []byte("HTTP/1.0 200 OK\r\n\r\n")
	HttpTunnelFail = []byte("HTTP/1.1 502 Bad Gateway\r\n\r\n")
	hasPort        = regexp.MustCompile(`:\d+$`)
)

// The tunnel proxy type. Implements http.Handler.
type TunnelHandler struct {
	Ctx *Context
}

// Create a tunnel handler
func NewTunnelHandler() *TunnelHandler {
	return &TunnelHandler{
		Ctx: NewContext(),
	}
}

// Standard net/http function. You can use it alone
func (tunnel *TunnelHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
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
	targetConn, err = tunnel.ConnectDial("tcp", targetAddr)
	if err != nil {
		ConnError(proxyClient)
		return
	}

	// The cascade proxy needs to forward the request
	if isCascadeProxy {
		// The cascading agent needs to send it as-is
		_ = req.Write(targetConn)
	} else {
		// Tell the client that the tunnel is ready
		_, _ = proxyClient.Write(HttpTunnelOk)
	}

	go func() {
		buf := make([]byte, 2048)
		_, _ = io.CopyBuffer(targetConn, proxyClient, buf)
		targetConn.Close()
		proxyClient.Close()
	}()
	buf := make([]byte, 2048)
	_, _ = io.CopyBuffer(proxyClient, targetConn, buf)
}

func (tunnel *TunnelHandler) ConnectDial(network, addr string) (net.Conn, error) {
	if tunnel.Ctx.Transport != nil && tunnel.Ctx.Transport.DialContext != nil {
		return tunnel.Ctx.Transport.DialContext(tunnel.Context(), network, addr)
	}
	return net.Dial(network, addr)
}

func (tunnel *TunnelHandler) Context() context.Context {
	if tunnel.Ctx.Context != nil {
		return tunnel.Ctx.Context
	}
	return context.Background()
}

func (tunnel *TunnelHandler) Transport() *http.Transport {
	return tunnel.Ctx.Transport
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
