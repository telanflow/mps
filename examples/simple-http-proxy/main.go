package main

import (
	"errors"
	"github.com/telanflow/mps"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"syscall"
)

// A simple http proxy server
func main() {
	quitSignChan := make(chan os.Signal)

	// create a http proxy server
	proxy := mps.NewHttpProxy()
	proxy.UseFunc(func(req *http.Request, ctx *mps.Context) (*http.Response, error) {
		log.Printf("[INFO] middleware -- %s %s", req.Method, req.URL)
		return ctx.Next(req)
	})

	reqGroup := proxy.OnRequest(mps.FilterHostMatches(regexp.MustCompile("^.*$")))
	reqGroup.DoFunc(func(req *http.Request, ctx *mps.Context) (*http.Request, *http.Response) {
		log.Printf("[INFO] req -- %s %s", req.Method, req.URL)
		return req, nil
	})

	respGroup := proxy.OnResponse()
	respGroup.DoFunc(func(resp *http.Response, err error, ctx *mps.Context) (*http.Response, error) {
		if err != nil {
			log.Printf("[ERRO] resp -- %s %v", ctx.Request.Method, err)
			return resp, err
		}

		log.Printf("[INFO] resp -- %d", resp.StatusCode)
		return resp, err
	})

	// Start server
	srv := &http.Server{
		Addr:    "localhost:8080",
		Handler: proxy,
	}
	go func() {
		log.Printf("HttpProxy started listen: http://%s", srv.Addr)
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		if err != nil {
			quitSignChan <- syscall.SIGKILL
			log.Fatalf("HttpProxy start fail: %v", err)
		}
	}()

	// quit signal
	signal.Notify(quitSignChan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)

	<-quitSignChan
	_ = srv.Close()
	log.Fatal("HttpProxy server stop!")
}
