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
		log.Printf("[INFO] middleware -- %s", req.URL)
		return ctx.Next(req)
	})

	reqGroup := proxy.OnRequest(mps.FilterHostMatches(regexp.MustCompile("^.*$")))
	reqGroup.DoFunc(func(req *http.Request, ctx *mps.Context) (*http.Request, *http.Response) {
		log.Printf("[INFO] req -- %s\n", req.URL)
		return req, nil
	})

	respGroup := proxy.OnResponse()
	respGroup.DoFunc(func(resp *http.Response, ctx *mps.Context) (*http.Response, error) {
		log.Printf("[INFO] resp -- %d\n", resp.StatusCode)
		return resp, nil
	})

	// Start server
	srv := &http.Server{
		Addr:    "127.0.0.1:8081",
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
