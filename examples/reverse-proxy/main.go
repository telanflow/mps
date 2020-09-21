package main

import (
	"errors"
	"github.com/telanflow/mps"
	"github.com/telanflow/mps/middleware"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
)

// A simple reverse proxy server
func main() {
	targetURL, _ := url.Parse("https://www.google.com")
	quitSignChan := make(chan os.Signal)

	// reverse proxy server
	proxy := mps.NewReverseHandler()
	proxy.UseFunc(middleware.SingleHostReverseProxy(targetURL))

	reqGroup := proxy.OnRequest()
	reqGroup.DoFunc(func(req *http.Request, ctx *mps.Context) (*http.Request, *http.Response) {
		log.Printf("[INFO] req -- %s %s", req.Method, req.Host)
		return req, nil
	})

	respGroup := proxy.OnResponse()
	respGroup.DoFunc(func(resp *http.Response, err error, ctx *mps.Context) (*http.Response, error) {
		if err != nil {
			log.Printf("[ERRO] resp -- %s %v", ctx.Request.Method, err)
			return nil, err
		}
		log.Printf("[INFO] resp -- %d", resp.StatusCode)

		// You have to reset Content-Length, if you change the Body.

		//var buf bytes.Buffer
		//buf.WriteString("body changed")
		//resp.Body = ioutil.NopCloser(&buf)
		//resp.ContentLength = int64(buf.Len())
		//resp.Header.Set("Content-Length", strconv.Itoa(buf.Len()))

		return resp, err
	})

	// started proxy server
	srv := http.Server{
		Addr:    "localhost:8080",
		Handler: proxy,
	}
	go func() {
		log.Printf("ReverseProxy started listen: http://%s", srv.Addr)
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		if err != nil {
			quitSignChan <- syscall.SIGKILL
			log.Fatalf("ReverseProxy start fail: %v", err)
		}
	}()

	// quit signal
	signal.Notify(quitSignChan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)

	<-quitSignChan
	_ = srv.Close()
	log.Fatal("ReverseProxy server stop!")
}
