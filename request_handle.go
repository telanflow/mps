package mps

import "net/http"

type RequestHandle interface {
	Handle(req *http.Request) (*http.Request, *http.Response)
}

type RequestHandleFunc func(req *http.Request) (*http.Request, *http.Response)

func (f RequestHandleFunc) Handle(req *http.Request) (*http.Request, *http.Response) {
	return f(req)
}
