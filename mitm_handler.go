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
	"io"
	"math/big"
	"math/rand"
	"net"
	"net/http"
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

type MitmHandler struct {
	Ctx         *Context
	Certificate tls.Certificate
	// CertContainer is certificate storage container
	CertContainer cert.Container
}

func NewMitmHandler() *MitmHandler {
	return &MitmHandler{
		Ctx: NewContext(),
		// default MPS Certificate
		Certificate:   cert.DefaultCertificate,
		CertContainer: cert.NewMemProvider(),
	}
}

func NewMitmHandlerWithCert(certFile, keyFile string) (*MitmHandler, error) {
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &MitmHandler{
		Ctx:           NewContext(),
		Certificate:   certificate,
		CertContainer: cert.NewMemProvider(),
	}, nil
}

func (mitm *MitmHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// get hijacker connection
	proxyClient, err := hijacker(w)
	if err != nil {
		http.Error(w, err.Error(), 502)
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
			//ctx.Warnf("Cannot handshake client %v %v", r.Host, err)
			return
		}
		defer rawClientTls.Close()

		clientTlsReader := bufio.NewReader(rawClientTls)
		for !isEof(clientTlsReader) {
			req, err := http.ReadRequest(clientTlsReader)
			if err != nil {
				//ctx.Warnf("Cannot read TLS request from mitm'd client %v %v", r.Host, err)
				break
			}

			// since we're converting the request, need to carry over the original connecting IP as well
			req.RemoteAddr = r.RemoteAddr

			if !httpsRegexp.MatchString(req.URL.String()) {
				req.URL, err = url.Parse("https://" + r.Host + req.URL.String())
			}
			if err != nil {
				//ctx.Warnf("Illegal URL %s", "https://"+r.Host+req.URL.Path)
				return
			}

			// Copying a Context preserves the Transport, Middleware
			ctx := mitm.Ctx.Copy()
			ctx.Request = req

			// In some cases it is not always necessary to remove the Proxy Header.
			// For example, cascade proxy
			if !mitm.Ctx.KeepHeader {
				removeProxyHeaders(req)
			}

			var resp *http.Response
			resp, err = ctx.Next(req)
			if err != nil {
				//ctx.Warnf("Cannot read TLS response from mitm'd server %v", err)
				return
			}
			defer resp.Body.Close()

			status := resp.Status
			statusCode := strconv.Itoa(resp.StatusCode) + " "
			if strings.HasPrefix(status, statusCode) {
				status = status[len(statusCode):]
			}

			// always use 1.1 to support chunked encoding
			if _, err := io.WriteString(rawClientTls, "HTTP/1.1"+" "+statusCode+status+"\r\n"); err != nil {
				//ctx.Warnf("Cannot write TLS response HTTP status from mitm'd client: %v", err)
				return
			}

			// Since we don't know the length of resp, return chunked encoded response
			// TODO: use a more reasonable scheme
			resp.Header.Del("Content-Length")
			resp.Header.Set("Transfer-Encoding", "chunked")

			// Force connection close otherwise chrome will keep CONNECT tunnel open forever
			resp.Header.Set("Connection", "close")

			err = resp.Header.Write(rawClientTls)
			if err != nil {
				//ctx.Warnf("Cannot write TLS response header from mitm'd client: %v", err)
				return
			}
			_, err = io.WriteString(rawClientTls, "\r\n")
			if err != nil {
				//ctx.Warnf("Cannot write TLS response header end from mitm'd client: %v", err)
				return
			}

			chunked := newChunkedWriter(rawClientTls)
			_, err = io.Copy(chunked, resp.Body)
			if err != nil {
				//ctx.Warnf("Cannot write TLS response body from mitm'd client: %v", err)
				return
			}
			if err := chunked.Close(); err != nil {
				//ctx.Warnf("Cannot write TLS chunked EOF from mitm'd client: %v", err)
				return
			}
			if _, err = io.WriteString(rawClientTls, "\r\n"); err != nil {
				//ctx.Warnf("Cannot write TLS response chunked trailer from mitm'd client: %v", err)
				return
			}
		}

	}()
}

func (mitm *MitmHandler) TLSConfigFromCA(host string) (*tls.Config, error) {
	host = stripPort(host)

	// Returned existing certificate for the host
	crt, err := mitm.CertContainer.Get(host)
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
	mitm.CertContainer.Set(host, crt)

	return &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{*crt},
	}, nil
}

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
		// SerialNumber 是 CA 颁布的唯一序列号，在此使用一个大随机数来代表它
		SerialNumber: big.NewInt(rand.Int63()),
		Issuer:       x509ca.Subject,
		// pkix.Name代表一个X.509识别名。只包含识别名的公共属性，额外的属性被忽略。
		Subject: pkix.Name{
			Organization: []string{"MPS untrusted MITM proxy Inc"},
		},
		NotBefore: time.Unix(0, 0),
		NotAfter:  time.Now().AddDate(20, 0, 0),
		// KeyUsage 与 ExtKeyUsage 用来表明该证书是用来做服务器认证的
		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		// 密钥扩展用途的序列
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
