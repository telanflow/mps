package mps

import (
	"bufio"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"github.com/telanflow/mps/cert"
	"github.com/telanflow/mps/pool"
	"io"
	"math/big"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	HttpMitmOk  = []byte("HTTP/1.0 200 OK\r\n\r\n")
	httpsRegexp = regexp.MustCompile("^https://")
)

// The Man-in-the-middle proxy type. Implements http.Handler.
type MitmHandler struct {
	Ctx         *Context
	BufferPool  httputil.BufferPool
	Certificate tls.Certificate
	// CertContainer is certificate storage container
	CertContainer cert.Container
}

// Create a MitmHandler, use default cert.
func NewMitmHandler() *MitmHandler {
	return &MitmHandler{
		Ctx:           NewContext(),
		BufferPool:    pool.DefaultBuffer,
		Certificate:   cert.DefaultCertificate,
		CertContainer: cert.NewMemProvider(),
	}
}

// Create a MitmHandler, use default cert.
func NewMitmHandlerWithContext(ctx *Context) *MitmHandler {
	return &MitmHandler{
		Ctx:           ctx,
		BufferPool:    pool.DefaultBuffer,
		Certificate:   cert.DefaultCertificate,
		CertContainer: cert.NewMemProvider(),
	}
}

// Create a MitmHandler with cert pem block
func NewMitmHandlerWithCert(certPEMBlock, keyPEMBlock []byte) (*MitmHandler, error) {
	certificate, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	if err != nil {
		return nil, err
	}
	return &MitmHandler{
		Ctx:           NewContext(),
		BufferPool:    pool.DefaultBuffer,
		Certificate:   certificate,
		CertContainer: cert.NewMemProvider(),
	}, nil
}

// Create a MitmHandler with cert file
func NewMitmHandlerWithCertFile(certFile, keyFile string) (*MitmHandler, error) {
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &MitmHandler{
		Ctx:           NewContext(),
		BufferPool:    pool.DefaultBuffer,
		Certificate:   certificate,
		CertContainer: cert.NewMemProvider(),
	}, nil
}

// Standard net/http function. You can use it alone
func (mitm *MitmHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// execution middleware
	ctx := mitm.Ctx.WithRequest(r)
	resp, err := ctx.Next(r)
	if err != nil && err != MethodNotSupportErr {
		if resp != nil {
			copyHeaders(rw.Header(), resp.Header, mitm.Ctx.KeepDestinationHeaders)
			rw.WriteHeader(resp.StatusCode)
			buf := mitm.buffer().Get()
			_, err = io.CopyBuffer(rw, resp.Body, buf)
			mitm.buffer().Put(buf)
		}
		return
	}

	// get hijacker connection
	proxyClient, err := hijacker(rw)
	if err != nil {
		http.Error(rw, err.Error(), 502)
		return
	}

	// this goes in a separate goroutine, so that the net/http server won't think we're
	// still handling the request even after hijacking the connection. Those HTTP CONNECT
	// request can take forever, and the server will be stuck when "closed".
	// TODO: Allow Server.Close() mechanism to shut down this connection as nicely as possible
	tlsConfig, err := mitm.TLSConfigFromCA(r.URL.Host)
	if err != nil {
		ConnError(proxyClient)
		return
	}

	_, _ = proxyClient.Write(HttpMitmOk)

	go func() {
		// TODO: cache connections to the remote website
		rawClientTls := tls.Server(proxyClient, tlsConfig)
		if err := rawClientTls.Handshake(); err != nil {
			ConnError(proxyClient)
			_ = rawClientTls.Close()
			return
		}
		defer rawClientTls.Close()

		clientTlsReader := bufio.NewReader(rawClientTls)
		for !isEof(clientTlsReader) {
			req, err := http.ReadRequest(clientTlsReader)
			if err != nil {
				break
			}

			// since we're converting the request, need to carry over the original connecting IP as well
			req.RemoteAddr = r.RemoteAddr

			if !httpsRegexp.MatchString(req.URL.String()) {
				req.URL, err = url.Parse("https://" + r.Host + req.URL.String())
			}
			if err != nil {
				return
			}

			var resp *http.Response

			// Copying a Context preserves the Transport, Middleware
			ctx := mitm.Ctx.WithRequest(req)
			resp, err = ctx.Next(req)
			if err != nil {
				return
			}

			status := resp.Status
			statusCode := strconv.Itoa(resp.StatusCode) + " "
			if strings.HasPrefix(status, statusCode) {
				status = status[len(statusCode):]
			}

			// always use 1.1 to support chunked encoding
			if _, err := io.WriteString(rawClientTls, "HTTP/1.1"+" "+statusCode+status+"\r\n"); err != nil {
				return
			}

			// Since we don't know the length of resp, return chunked encoded response
			resp.Header.Set("Transfer-Encoding", "chunked")

			// Force connection close otherwise chrome will keep CONNECT tunnel open forever
			resp.Header.Set("Connection", "close")

			err = resp.Header.Write(rawClientTls)
			if err != nil {
				resp.Body.Close()
				return
			}
			_, err = io.WriteString(rawClientTls, "\r\n")
			if err != nil {
				resp.Body.Close()
				return
			}

			chunked := newChunkedWriter(rawClientTls)

			buf := mitm.buffer().Get()
			_, err = io.CopyBuffer(chunked, resp.Body, buf)
			mitm.buffer().Put(buf)
			if err != nil {
				resp.Body.Close()
				return
			}

			// closed response body
			resp.Body.Close()

			if err := chunked.Close(); err != nil {
				return
			}
			if _, err = io.WriteString(rawClientTls, "\r\n"); err != nil {
				return
			}
		}
	}()
}

// Use registers an Middleware to proxy
func (mitm *MitmHandler) Use(middleware ...Middleware) {
	mitm.Ctx.Use(middleware...)
}

// UseFunc registers an MiddlewareFunc to proxy
func (mitm *MitmHandler) UseFunc(fus ...MiddlewareFunc) {
	mitm.Ctx.UseFunc(fus...)
}

// OnRequest filter requests through Filters
func (mitm *MitmHandler) OnRequest(filters ...Filter) *ReqFilterGroup {
	return &ReqFilterGroup{ctx: mitm.Ctx, filters: filters}
}

// OnResponse filter response through Filters
func (mitm *MitmHandler) OnResponse(filters ...Filter) *RespFilterGroup {
	return &RespFilterGroup{ctx: mitm.Ctx, filters: filters}
}

// Get buffer pool
func (mitm *MitmHandler) buffer() httputil.BufferPool {
	if mitm.BufferPool != nil {
		return mitm.BufferPool
	}
	return pool.DefaultBuffer
}

// Get cert.Container instance
func (mitm *MitmHandler) certContainer() cert.Container {
	if mitm.CertContainer != nil {
		return mitm.CertContainer
	}
	return cert.DefaultMemProvider
}

// Transport
func (mitm *MitmHandler) Transport() *http.Transport {
	return mitm.Ctx.Transport
}

func (mitm *MitmHandler) TLSConfigFromCA(host string) (*tls.Config, error) {
	host = stripPort(host)

	// Returned existing certificate for the host
	crt, err := mitm.certContainer().Get(host)
	if err == nil && crt != nil {
		return &tls.Config{
			InsecureSkipVerify: true,
			Certificates:       []tls.Certificate{*crt},
		}, nil
	}

	// Issue a certificate for host
	crt, err = signHost(mitm.Certificate, []string{host})
	if err != nil {
		err = fmt.Errorf("cannot sign host certificate with provided CA: %v", err)
		return nil, err
	}

	// Set certificate to container
	_ = mitm.certContainer().Set(host, crt)

	return &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{*crt},
	}, nil
}

// sign host
func signHost(ca tls.Certificate, hosts []string) (cert *tls.Certificate, err error) {
	// Use the provided ca for certificate generation.
	var x509ca *x509.Certificate
	x509ca, err = x509.ParseCertificate(ca.Certificate[0])
	if err != nil {
		return
	}

	var random CounterEncryptorRand
	random, err = NewCounterEncryptorRand(ca.PrivateKey, hashHosts(hosts))
	if err != nil {
		return
	}

	var pk crypto.Signer
	switch ca.PrivateKey.(type) {
	case *rsa.PrivateKey:
		pk, err = rsa.GenerateKey(&random, 2048)
	case *ecdsa.PrivateKey:
		pk, err = ecdsa.GenerateKey(elliptic.P256(), &random)
	default:
		err = fmt.Errorf("unsupported key type %T", ca.PrivateKey)
	}
	if err != nil {
		return
	}

	// certificate template
	tpl := x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63()),
		Issuer:       x509ca.Subject,
		Subject: pkix.Name{
			Organization: []string{"MPS untrusted MITM proxy Inc"},
		},
		NotBefore:             time.Unix(0, 0),
		NotAfter:              time.Now().AddDate(20, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		EmailAddresses:        x509ca.EmailAddresses,
	}

	total := len(hosts)
	for i := 0; i < total; i++ {
		if ip := net.ParseIP(hosts[i]); ip != nil {
			tpl.IPAddresses = append(tpl.IPAddresses, ip)
		} else {
			tpl.DNSNames = append(tpl.DNSNames, hosts[i])
			tpl.Subject.CommonName = hosts[i]
		}
	}

	var der []byte
	der, err = x509.CreateCertificate(&random, &tpl, x509ca, pk.Public(), ca.PrivateKey)
	if err != nil {
		return
	}

	cert = &tls.Certificate{
		Certificate: [][]byte{der, ca.Certificate[0]},
		PrivateKey:  pk,
	}
	return
}

func stripPort(s string) string {
	ix := strings.IndexRune(s, ':')
	if ix == -1 {
		return s
	}
	return s[:ix]
}

func hashHosts(lst []string) []byte {
	c := make([]string, len(lst))
	copy(c, lst)
	sort.Strings(c)
	h := sha1.New()
	h.Write([]byte(strings.Join(c, ",")))
	return h.Sum(nil)
}

// cloneTLSConfig returns a shallow clone of cfg, or a new zero tls.Config if
// cfg is nil. This is safe to call even if cfg is in active use by a TLS
// client or server.
func cloneTLSConfig(cfg *tls.Config) *tls.Config {
	if cfg == nil {
		return &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	return cfg.Clone()
}

func isEof(r *bufio.Reader) bool {
	_, err := r.Peek(1)
	if err == io.EOF {
		return true
	}
	return false
}
