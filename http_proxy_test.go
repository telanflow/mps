package mps

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func NewTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		query := req.URL.Query()
		text := []byte("hello world")
		if query.Get("text") != "" {
			text = []byte(query.Get("text"))
		}

		rw.Header().Set("Server", "MPS proxy server")
		_, _ = rw.Write(text)
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
		resp, err := ctx.Next(req)
		if err != nil {
			return nil, err
		}

		var buf bytes.Buffer
		buf.WriteString("middleware")
		resp.Body = ioutil.NopCloser(&buf)

		//
		// You have to reset Content-Length, if you change the Body.
		//resp.ContentLength = int64(buf.Len())
		//resp.Header.Set("Content-Length", strconv.Itoa(buf.Len()))

		return resp, nil
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
	asserts.Equal(string(body), "middleware")
}
