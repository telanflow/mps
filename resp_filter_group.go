package mps

import (
	"net/http"
)

type RespFilterGroup struct {
	ctx     *Context
	filters []Filter
}

func (cond *RespFilterGroup) DoFunc(fn func(resp *http.Response, err error, ctx *Context) (*http.Response, error)) {
	cond.Do(ResponseHandleFunc(fn))
}

func (cond *RespFilterGroup) Do(h ResponseHandle) {
	cond.ctx.UseFunc(func(req *http.Request, ctx *Context) (*http.Response, error) {
		total := len(cond.filters)
		for i := 0; i < total; i++ {
			if !cond.filters[i].Match(req) {
				return ctx.Next(req)
			}
		}
		resp, err := ctx.Next(req)
		return h.Handle(resp, err, ctx)
	})
}
