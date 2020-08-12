package mps

import (
	"net/http"
)

// ReqCondition is a request condition group
type ReqFilterGroup struct {
	ctx     *Context
	filters []Filter
}

func (cond *ReqFilterGroup) DoFunc(fn func(req *http.Request, ctx *Context) (*http.Request, *http.Response)) {
	cond.Do(RequestHandleFunc(fn))
}

func (cond *ReqFilterGroup) Do(h RequestHandle) {
	cond.ctx.UseFunc(func(req *http.Request, ctx *Context) (*http.Response, error) {
		total := len(cond.filters)
		for i := 0; i < total; i++ {
			if !cond.filters[i].Match(req) {
				return ctx.Next(req)
			}
		}

		req, resp := h.Handle(req, ctx)
		if resp != nil {
			return resp, nil
		}

		return ctx.Next(req)
	})
}
