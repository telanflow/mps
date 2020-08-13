package mps

import (
	"bufio"
	"context"
	"github.com/telanflow/mps/pool"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// The websocket proxy type. Implements http.Handler.
type WebsocketHandler struct {
	Ctx        *Context
	BufferPool httputil.BufferPool
}

// Create a websocket handler
func NewWebsocketHandler() *WebsocketHandler {
	return &WebsocketHandler{
		Ctx:        NewContext(),
		BufferPool: pool.DefaultBuffer,
	}
}

// Create a tunnel handler with Context
func NewWebsocketHandlerWithContext(ctx *Context) *WebsocketHandler {
	return &WebsocketHandler{
		Ctx:        ctx,
		BufferPool: pool.DefaultBuffer,
	}
}

// Standard net/http function. You can use it alone
func (ws *WebsocketHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Whether to upgrade to Websocket
	if !isWebSocketRequest(req) {
		return
	}

	// hijacker connection
	clientConn, err := hijacker(rw)
	if err != nil {
		http.Error(rw, err.Error(), 502)
		return
	}

	var (
		u          *url.URL
		targetAddr = hostAndPort(req.URL.Host)
	)
	if ws.Ctx.Transport != nil && ws.Ctx.Transport.Proxy != nil {
		u, err = ws.Ctx.Transport.Proxy(req)
		if err != nil {
			ConnError(clientConn)
			return
		}
		if u != nil {
			// connect addr eg. "localhost:443"
			targetAddr = hostAndPort(u.Host)
		}
	}

	targetConn, err := ws.ConnectDial("tcp", targetAddr)
	if err != nil {
		return
	}
	defer targetConn.Close()

	// Perform handshake
	// write handshake request to target
	err = req.Write(targetConn)
	if err != nil {
		return
	}

	// Read handshake response from target
	targetReader := bufio.NewReader(targetConn)
	resp, err := http.ReadResponse(targetReader, req)
	if err != nil {
		return
	}

	// Proxy handshake back to client
	err = resp.Write(clientConn)
	if err != nil {
		return
	}

	// Proxy ws connection
	go func() {
		buf := ws.buffer().Get()
		_, _ = io.CopyBuffer(targetConn, clientConn, buf)
		ws.buffer().Put(buf)
		_ = clientConn.Close()
	}()
	buf := ws.buffer().Get()
	_, _ = io.CopyBuffer(clientConn, targetConn, buf)
	ws.buffer().Put(buf)
}

func (ws *WebsocketHandler) ConnectDial(network, addr string) (net.Conn, error) {
	if ws.Ctx.Transport != nil && ws.Ctx.Transport.DialContext != nil {
		return ws.Ctx.Transport.DialContext(ws.context(), network, addr)
	}
	return net.DialTimeout(network, addr, 30*time.Second)
}

// context returned a context.Context
func (ws *WebsocketHandler) context() context.Context {
	if ws.Ctx.Context != nil {
		return ws.Ctx.Context
	}
	return context.Background()
}

// buffer returned a httputil.BufferPool
func (ws *WebsocketHandler) buffer() httputil.BufferPool {
	if ws.BufferPool != nil {
		return ws.BufferPool
	}
	return pool.DefaultBuffer
}

// isWebSocketRequest to upgrade to a Websocket request
func isWebSocketRequest(req *http.Request) bool {
	return headerContains(req.Header, "Connection", "upgrade") &&
		headerContains(req.Header, "Upgrade", "websocket")
}

func headerContains(header http.Header, name string, value string) bool {
	for _, v := range header[name] {
		for _, s := range strings.Split(v, ",") {
			if strings.EqualFold(value, strings.TrimSpace(s)) {
				return true
			}
		}
	}
	return false
}
