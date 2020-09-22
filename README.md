<h1 align="center">
  <br>MPS<br>
</h1>

English | [🇨🇳中文](README_ZH.md)

## 📖 Introduction
![MPS](https://github.com/telanflow/mps/workflows/MPS/badge.svg)
![stars](https://img.shields.io/github/stars/telanflow/mps)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/telanflow/mps)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/telanflow/mps)
[![license](https://img.shields.io/github/license/telanflow/mps)](https://github.com/telanflow/mps/LICENSE)

MPS (middle-proxy-server) is an high-performance middle proxy library. support HTTP, HTTPS, Websocket, ForwardProxy, ReverseProxy, TunnelProxy, MitmProxy.

## 🚀 Features
- [X] Http Proxy
- [X] Https Proxy
- [X] Forward Proxy
- [X] Reverse Proxy
- [X] Tunnel Proxy
- [X] Mitm Proxy (Man-in-the-middle) 
- [X] WekSocket Proxy

## 🧰 Install
```
go get -u github.com/telanflow/mps
```

## 🛠 How to use
A simple proxy service

```go
package main

import (
    "github.com/telanflow/mps"
    "log"
    "net/http"
)

func main() {
    proxy := mps.NewHttpProxy()
    log.Fatal(http.ListenAndServe(":8080", proxy))
}
```

More [examples](https://github.com/telanflow/mps/tree/master/examples)

## 🧬 Middleware
Middleware can intercept requests and responses. 
we have several middleware implementations built in, including [BasicAuth](https://github.com/telanflow/mps/tree/master/middleware)

```go
func main() {
    proxy := mps.NewHttpProxy()
    
    proxy.Use(mps.MiddlewareFunc(func(req *http.Request, ctx *mps.Context) (*http.Response, error) {
        log.Printf("[INFO] middleware -- %s %s", req.Method, req.URL)
        return ctx.Next(req)
    }))
    
    proxy.UseFunc(func(req *http.Request, ctx *mps.Context) (*http.Response, error) {
        log.Printf("[INFO] middleware -- %s %s", req.Method, req.URL)
        resp, err := ctx.Next(req)
        if err != nil {
            return nil, err
        }
        log.Printf("[INFO] resp -- %d", resp.StatusCode)
        return resp, err
    })
    
    log.Fatal(http.ListenAndServe(":8080", proxy))
}
```

## ♻️ Filters
Filters can filter requests and responses for unified processing.
It is based on middleware implementation.

```go
func main() {
    proxy := mps.NewHttpProxy()
    
    // request Filter Group
    reqGroup := proxy.OnRequest(mps.FilterHostMatches(regexp.MustCompile("^.*$")))
    reqGroup.DoFunc(func(req *http.Request, ctx *mps.Context) (*http.Request, *http.Response) {
        log.Printf("[INFO] req -- %s %s", req.Method, req.URL)
        return req, nil
    })
    
    // response Filter Group
    respGroup := proxy.OnResponse()
    respGroup.DoFunc(func(resp *http.Response, err error, ctx *mps.Context) (*http.Response, error) {
        if err != nil {
            log.Printf("[ERRO] resp -- %s %v", ctx.Request.Method, err)
            return nil, err
        }
    
        log.Printf("[INFO] resp -- %d", resp.StatusCode)
        return resp, err
    })
    
    log.Fatal(http.ListenAndServe(":8080", proxy))
}
```

## 📄 License
Source code in `MPS` is available under the [BSD 3 License](/LICENSE).
