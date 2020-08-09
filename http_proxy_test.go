package mps

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNewHttpProxy(t *testing.T) {
	proxy := NewHttpProxy()
	srv := httptest.NewServer(proxy)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, "http://httpbin.org/get", nil)
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
