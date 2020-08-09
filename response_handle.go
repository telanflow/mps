package mps

import "net/http"

type ResponseHandle interface {
	Handle(resp *http.Response) (*http.Response, error)
}

type ResponseHandleFunc func(resp *http.Response) (*http.Response, error)

func (f ResponseHandleFunc) Handle(resp *http.Response) (*http.Response, error) {
	return f(resp)
}
