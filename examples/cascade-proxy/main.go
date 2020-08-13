package main

import (
	"github.com/telanflow/mps"
	"github.com/telanflow/mps/middleware"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

// A simple example of cascading proxy.
// It implements BasicAuth
func main() {
	// endPoint server
	go http.ListenAndServe("localhost:9990", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("successful endPoint server"))
	}))

	// proxy server 1
	proxy1 := mps.NewHttpProxy()
	proxy1.Ctx.KeepProxyHeaders = true
	proxy1.Use(middleware.BasicAuth("mps_realm_1", func(username, password string) bool {
		return username == "foo_1" && password == "bar_1"
	}))
	go http.ListenAndServe("localhost:9991", proxy1)

	// proxy server 2
	proxy2 := mps.NewHttpProxy()
	proxy2.Ctx.KeepProxyHeaders = true
	proxy2.Use(middleware.BasicAuth("mps_realm_2", func(username, password string) bool {
		return username == "foo_2" && password == "bar_2"
	}))
	proxy2.Transport().Proxy = func(req *http.Request) (*url.URL, error) {
		middleware.SetBasicAuth(req, "foo_1", "bar_1")
		return url.Parse("http://localhost:9991")
	}
	go http.ListenAndServe("localhost:9992", proxy2)

	// wait proxy server started
	time.Sleep(2 * time.Second)

	// send request
	// request ==> proxy2 ==> proxy1 ==> http://localhost:9990
	// response <== proxy2 <== proxy1 <== http://localhost:9990
	req, _ := http.NewRequest(http.MethodGet, "http://localhost:9990/", nil)
	http.DefaultClient.Transport = &http.Transport{
		Proxy: func(r *http.Request) (*url.URL, error) {
			middleware.SetBasicAuth(r, "foo_2", "bar_2")
			return url.Parse("http://localhost:9992")
		},
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	log.Println(resp.Header)
	log.Println(string(body))
}
