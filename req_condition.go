package mps

import (
	"net/http"
)

type ReqCondition struct {
	ctx     *Context
	filters []Filter
}

func (cond *ReqCondition) DoFunc(fn func(req *http.Request) (*http.Request, *http.Response)) {
	cond.Do(RequestHandleFunc(fn))
}

func (cond *ReqCondition) Do(fn RequestHandle) {
	cond.ctx.UseFunc(func(req *http.Request, ctx *Context) (*http.Response, error) {
		total := len(cond.filters)
		for i := 0; i < total; i++ {
			if !cond.filters[i].Match(req) {
				return ctx.Next(req)
			}
		}

		req, resp := fn.Handle(req)
		if resp != nil {
			return resp, nil
		}

		return ctx.Next(req)
	})
}
