package mps

import (
	"net/http"
	"regexp"
)

type Filter interface {
	Match(expr string) bool
}

type FilterFunc func(expr string) bool

func (f FilterFunc) Match(expr string) bool {
	return f(expr)
}

// 匹配域名
var MatchIsHost = func(expr string, req *http.Request) Filter {
	exp, err := regexp.Compile(expr)
	if err != nil {
		panic(err)
	}
	return FilterFunc(func(expr string) bool {
		return exp.MatchString(req.Host)
	})
}

type ReqHandler interface {
	Handler(ctx *Context)
}
