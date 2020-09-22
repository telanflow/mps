<h1 align="center">
  <br>MPS<br>
</h1>

[English](README.md) | 🇨🇳中文

## 📖 介绍
![MPS](https://github.com/telanflow/mps/workflows/MPS/badge.svg)
![stars](https://img.shields.io/github/stars/telanflow/mps)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/telanflow/mps)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/telanflow/mps)
[![license](https://img.shields.io/github/license/telanflow/mps)](https://github.com/telanflow/mps/LICENSE)

MPS 是一个高性能的中间代理扩展库，支持 HTTP、HTTPS、Websocket、正向代理、反向代理、隧道代理、中间人代理 等代理方式。

## 🚀 特性
- [X] Http代理
- [X] Https代理
- [X] 正向代理
- [X] 反向代理
- [X] 隧道代理
- [X] 中间人代理 (MITM)
- [X] WekSocket代理

## 🧰 安装
```
go get -u github.com/telanflow/mps
```

## 🛠 如何使用
一个简单的HTTP代理服务

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

更多 [范例](https://github.com/telanflow/mps/tree/master/examples)

## 🧬 中间件
中间件可以拦截请求和响应，我们内置实现了多个中间件，包括 [BasicAuth](https://github.com/telanflow/mps/tree/master/middleware)

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

## ♻️ 过滤器
过滤器可以对请求和响应进行筛选，统一进行处理。
它基于中间件实现。

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

## 📄 开源许可
`MPS`中的源代码在[BSD 3 License](/LICENSE)下可用。
