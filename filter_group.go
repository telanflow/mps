package mps

import "net/http"

type FilterGroup interface {
	Handle()
}

// ReqCondition is a request filter group
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

		req, resp := h.HandleRequest(req, ctx)
		if resp != nil {
			return resp, nil
		}

		return ctx.Next(req)
	})
}

// ReqCondition is a response filter group
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
		return h.HandleResponse(resp, err, ctx)
	})
}
