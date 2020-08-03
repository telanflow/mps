package mps

import "net/http"

type Middleware func(req *http.Request, resp *http.Response)

type a http.HandlerFunc