package main

import (
	"net/http"

	"github.com/elazarl/goproxy"
)

func initProxy(proxy *goproxy.ProxyHttpServer) {
	cond := proxy.OnRequest(goproxy.DstHostIs("userstream.twitter.com"))
	cond.HandleConnect(goproxy.AlwaysMitm)
	cond.DoFunc(ProxyAuth(handleProxyStreaming))

	cond = proxy.OnRequest(goproxy.DstHostIs("api.twitter.com"))
	cond.HandleConnect(goproxy.AlwaysMitm)
	cond.DoFunc(ProxyAuth(handleProxyApi))

	proxy.OnRequest().HandleConnect(
		goproxy.FuncHttpsHandler(
			func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
				return goproxy.OkConnect, host
			},
		),
	)

	proxy.NonproxyHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})
}

func handleProxyApi(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {

	return req, ctx.Resp
}

func handleProxyStreaming(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	return req, ctx.Resp
}
