package mps

import "net/http"

type ResponseHandle interface {
	Handle(resp *http.Response) *http.Response
}

type ResponseHandleFunc func(resp *http.Response) *http.Response

func (f ResponseHandleFunc) Handle(resp *http.Response) *http.Response {
	return f(resp)
}
