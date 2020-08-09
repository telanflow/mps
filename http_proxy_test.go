package mps

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func NewTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Server", "MPS proxy server")
		rw.Write([]byte("hello world"))
	}))
}

func TestNewHttpProxy(t *testing.T) {
	asserts := assert.New(t)

	srv := NewTestServer()
	defer srv.Close()

	proxy := NewHttpProxy()
	proxySrv := httptest.NewServer(proxy)
	defer proxySrv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	http.DefaultClient.Transport = &http.Transport{
		Proxy: func(r *http.Request) (*url.URL, error) {
			return url.Parse(srv.URL)
		},
	}

	resp, err := http.DefaultClient.Do(req)
	asserts.Equal(err, nil, "err should be equal nil")

	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	asserts.Equal(resp.StatusCode, 200, "statusCode should be equal 200")
	asserts.Equal(int64(len(body)), resp.ContentLength)
}

func TestMiddlewareFunc(t *testing.T) {
	proxy := NewHttpProxy()
	proxy.UseFunc(func(req *http.Request, ctx *Context) (*http.Response, error) {
		log.Println(req.URL.String())
		return ctx.Next(req)
	})
	srv := httptest.NewServer(proxy)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, "https://httpbin.org/get", nil)
	http.DefaultClient.Transport = &http.Transport{
		Proxy: func(r *http.Request) (*url.URL, error) {
			return url.Parse(srv.URL)
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
