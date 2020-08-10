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

func HttpGet(rawurl string, proxy func(r *http.Request) (*url.URL, error)) (*http.Response, error) {
	req, _ := http.NewRequest(http.MethodGet, rawurl, nil)
	http.DefaultClient.Transport = &http.Transport{
		Proxy: proxy,
	}
	return http.DefaultClient.Do(req)
}

func TestNewHttpProxy(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	proxy := NewHttpProxy()
	proxySrv := httptest.NewServer(proxy)
	defer proxySrv.Close()

	resp, err := HttpGet(srv.URL, func(r *http.Request) (*url.URL, error) {
		return url.Parse(proxySrv.URL)
	})
	if err != nil {
		t.Fatal(err)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	asserts := assert.New(t)
	asserts.Equal(resp.StatusCode, 200, "statusCode should be equal 200")
	asserts.Equal(int64(len(body)), resp.ContentLength)
}

func TestMiddlewareFunc(t *testing.T) {
	// target server
	srv := NewTestServer()
	defer srv.Close()

	// proxy server
	proxy := NewHttpProxy()
	// use Middleware
	proxy.UseFunc(func(req *http.Request, ctx *Context) (*http.Response, error) {
		log.Println(req.URL.String())
		return ctx.Next(req)
	})
	proxySrv := httptest.NewServer(proxy)
	defer proxySrv.Close()

	// send request
	resp, err := HttpGet(srv.URL, func(r *http.Request) (*url.URL, error) {
		return url.Parse(proxySrv.URL)
	})
	if err != nil {
		t.Fatal(err)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	asserts := assert.New(t)
	asserts.Equal(resp.StatusCode, 200)
	asserts.Equal(int64(len(body)), resp.ContentLength)
	log.Println(string(body))
}
