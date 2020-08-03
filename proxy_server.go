package mps

import (
	"bufio"
	"crypto/tls"
	"io"
	"net/http"
)

type ProxyServer struct {
	middleware []Middleware
	Transport *http.Transport
	KeepHeader bool
	// KeepDestinationHeaders indicates the proxy should retain any headers present in the http.Response before proxying
	KeepDestinationHeaders bool
}

func NewProxyServer() *ProxyServer {
	return &ProxyServer{
		middleware: nil,
		Transport:  &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy: http.ProxyFromEnvironment,
		},
	}
}

func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handlerHttps(w, r)
		return
	}

	p.handlerHttp(w, r)
}

func (p *ProxyServer) handlerHttp(w http.ResponseWriter, r *http.Request) {
	var err error

	if !p.KeepHeader {
		removeProxyHeaders(r)
	}

	resp, err := p.Transport.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	origBody := resp.Body
	defer origBody.Close()

	// http.ResponseWriter will take care of filling the correct response length
	// Setting it now, might impose wrong value, contradicting the actual new
	// body the user returned.
	// We keep the original body to remove the header only if things changed.
	// This will prevent problems with HEAD requests where there's no body, yet,
	// the Content-Length header should be set.
	if origBody != resp.Body {
		resp.Header.Del("Content-Length")
	}
	copyHeaders(w.Header(), resp.Header, p.KeepDestinationHeaders)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	resp.Body.Close()
}

func (p *ProxyServer) handlerHttps(w http.ResponseWriter, r *http.Request) {
	hij, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Not a hijacker?", 500)
		return
	}

	proxyClientConn, _, e := hij.Hijack()
	if e != nil {
		http.Error(w, "Cannot hijack connection " + e.Error(), 500)
		return
	}


}

func removeProxyHeaders(r *http.Request) {
	r.RequestURI = "" // this must be reset when serving a request with the client
	// If no Accept-Encoding header exists, Transport will add the headers it can accept
	// and would wrap the response body with the relevant reader.
	r.Header.Del("Accept-Encoding")
	// curl can add that, see
	// https://jdebp.eu./FGA/web-proxy-connection-header.html
	r.Header.Del("Proxy-Connection")
	r.Header.Del("Proxy-Authenticate")
	r.Header.Del("Proxy-Authorization")
	// Connection, Authenticate and Authorization are single hop Header:
	// http://www.w3.org/Protocols/rfc2616/rfc2616.txt
	// 14.10 Connection
	//   The Connection general-header field allows the sender to specify
	//   options that are desired for that particular connection and MUST NOT
	//   be communicated by proxies over further connections.
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

func isEof(r *bufio.Reader) bool {
	_, err := r.Peek(1)
	if err == io.EOF {
		return true
	}
	return false
}