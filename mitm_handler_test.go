package mps

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/telanflow/mps/cert"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNewMitmHandler(t *testing.T) {
	mitm := NewMitmHandler()
	mitmSrv := httptest.NewServer(mitm)
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

	log.Println(err)
	log.Println(resp.Status)
	log.Println(string(body))
}
