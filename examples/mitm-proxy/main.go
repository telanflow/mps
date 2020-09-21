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

// A simple mitm proxy server
func main() {
	quitSignChan := make(chan os.Signal)

	// create proxy server
	proxy := mps.NewHttpProxy()

	// Load cert file
	// The Connect request is processed using MitmHandler
	mitmHandler, err := mps.NewMitmHandlerWithCertFile(proxy.Ctx, "./examples/mitm-proxy/ca.crt", "./examples/mitm-proxy/ca.key")
	if err != nil {
		log.Panic(err)
	}
	proxy.HandleConnect = mitmHandler

	// Middleware
	proxy.UseFunc(func(req *http.Request, ctx *mps.Context) (*http.Response, error) {
		log.Printf("[INFO] middleware -- %s %s", req.Method, req.URL)
		return ctx.Next(req)
	})

	// Filter
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

		// You have to reset Content-Length, if you change the Body.
		
		//var buf bytes.Buffer
		//buf.WriteString("body changed")
		//resp.Body = ioutil.NopCloser(&buf)
		//resp.ContentLength = int64(buf.Len())
		//resp.Header.Set("Content-Length", strconv.Itoa(buf.Len()))

		return resp, err
	})

	// Started proxy server
	srv := http.Server{
		Addr:    "localhost:8080",
		Handler: proxy,
	}
	go func() {
		log.Printf("MitmProxy started listen: http://%s", srv.Addr)
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		if err != nil {
			quitSignChan <- syscall.SIGKILL
			log.Fatalf("MitmProxy start fail: %v", err)
		}
	}()

	// quit signal
	signal.Notify(quitSignChan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)

	<-quitSignChan
	_ = srv.Close()
	log.Fatal("MitmProxy server stop!")
}
