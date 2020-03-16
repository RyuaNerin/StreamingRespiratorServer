package main

import (
	"net/http"

	"github.com/elazarl/goproxy"
)

func handleStreaming(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	return req, ctx.Resp
}
