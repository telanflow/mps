package mps

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/stretchr/testify/assert"
	"github.com/telanflow/mps/cert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNewMitmHandler(t *testing.T) {
	mitmHandler := NewMitmHandler()
	mitmSrv := httptest.NewServer(mitmHandler)
	defer mitmSrv.Close()

	clientCertPool := x509.NewCertPool()
	ok := clientCertPool.AppendCertsFromPEM([]byte(cert.CertPEM))
	if !ok {
		panic("failed to parse root certificate")
	}

	req, _ := http.NewRequest(http.MethodGet, "https://httpbin.org/get", nil)
	http.DefaultClient.Transport = &http.Transport{
		Proxy: func(r *http.Request) (*url.URL, error) {
			return url.Parse(mitmSrv.URL)
		},
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert.DefaultCertificate},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			RootCAs:      clientCertPool,
		},
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	asserts := assert.New(t)
	asserts.Equal(resp.StatusCode, 200, "response status code not equal 200")
	asserts.Equal(int64(len(body)), resp.ContentLength)
}
