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

// A simple BasicAuth example
func main() {
	// endPoint server
	go http.ListenAndServe("localhost:8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Basic Authentication
		usr, pwd, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(401)
			w.Write([]byte("401 Authentication Required"))
			return
		}
		if usr != "test" || pwd != "test" {
			w.WriteHeader(401)
			w.Write([]byte("401 Authentication Required"))
			return
		}
		w.Write([]byte("successful endPoint"))
	}))

	// proxy server
	proxy := mps.NewHttpProxy()
	// proxy BasicAuth
	proxy.Use(middleware.BasicAuth("mps_realm", func(username, password string) bool {
		return username == "mps" && password == "mps"
	}))
	proxy.UseFunc(func(req *http.Request, ctx *mps.Context) (*http.Response, error) {
		// set endPoint BasicAuth
		// Or you can set the endPoint BasicAuth on the client
		req.SetBasicAuth("test", "test")
		return ctx.Next(req)
	})
	go http.ListenAndServe("localhost:8081", proxy)

	// wait proxy started
	time.Sleep(2 * time.Second)

	// send request
	// request ==> proxy ==> http://localhost:8080
	// response <== proxy <== http://localhost:8080
	request, _ := http.NewRequest(http.MethodGet, "http://localhost:8080", nil)
	http.DefaultClient.Transport = &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			// set proxy server BasicAuth
			middleware.SetBasicAuth(req, "mps", "mps")

			// set endPoint BasicAuth
			// Or you can set the endPoint to BasicAuth on the proxy server
			//req.SetBasicAuth("test", "test")

			return url.Parse("http://localhost:8081")
		},
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	log.Println(string(body))
}
