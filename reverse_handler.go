package mps

import "net/http"

// ReverseHandler is a reverse proxy server implementation
type ReverseHandler struct {
	Ctx *Context
}

func NewReverseHandler() *ReverseHandler {
	return &ReverseHandler{
		Ctx: NewContext(),
	}
}

func (reverse *ReverseHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

}
