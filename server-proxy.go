package main

import (
	"encoding/base64"
	"net/http"
	"strings"

	"gopkg.in/elazarl/goproxy.v1"
)

const (
	proxyAuthorizationHeader = "Proxy-Authorization"
)

func newProxyServer() *goproxy.ProxyHttpServer {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Logger = logger
	proxy.Verbose = verbose

	cond := proxy.OnRequest(goproxy.DstHostIs("userstream.twitter.com"))
	cond.HandleConnect(proxyAuthConnect(goproxy.MitmConnect))
	cond.DoFunc(handleProxyStreaming)

	cond = proxy.OnRequest(goproxy.DstHostIs("api.twitter.com"))
	cond.HandleConnect(proxyAuthConnect(goproxy.MitmConnect))
	cond.DoFunc(handleProxyApi)

	cond = proxy.OnRequest()
	cond.HandleConnect(proxyAuthConnect(goproxy.OkConnect))
	cond.DoFunc(proxyAuth)

	proxy.NonproxyHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	return proxy
}

func splitProxyAuthorizationHeader(req *http.Request) (id string, pw string, ok bool) {
	authheader := strings.SplitN(req.Header.Get(proxyAuthorizationHeader), " ", 2)
	req.Header.Del(proxyAuthorizationHeader)
	if len(authheader) != 2 || authheader[0] != "Basic" {
		return
	}
	userpassraw, err := base64.StdEncoding.DecodeString(authheader[1])
	if err != nil {
		return
	}
	userpass := strings.SplitN(string(userpassraw), ":", 2)
	if len(userpass) != 2 {
		return
	}
	return userpass[0], userpass[1], true
}

func proxyAuthConnect(action *goproxy.ConnectAction) goproxy.FuncHttpsHandler {
	return func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		id, pw, ok := splitProxyAuthorizationHeader(ctx.Req)
		if !ok {
			ctx.Resp = newProxyResponseUnauthorized(ctx.Req, http.StatusProxyAuthRequired)
			return goproxy.RejectConnect, host
		}
		if id != authId || pw != authPw {
			ctx.Resp = newProxyResponseUnauthorized(ctx.Req, http.StatusUnauthorized)
			return goproxy.RejectConnect, host
		}

		return action, host
	}
}

func proxyAuth(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	id, pw, ok := splitProxyAuthorizationHeader(ctx.Req)
	if !ok {
		return req, newProxyResponseUnauthorized(req, http.StatusProxyAuthRequired)
	}
	if id != authId || pw != authPw {
		return req, newProxyResponseUnauthorized(req, http.StatusUnauthorized)
	}

	return req, nil
}

func newProxyResponseUnauthorized(req *http.Request, statusCode int) *http.Response {
	res := newResponse(req, statusCode)
	res.Header = http.Header{
		"Proxy-Authenticate": []string{"Basic realm=\"Access to Streamning-Respirator\""},
		"Proxy-Connection":   []string{"close"},
	}
	return res
}

func handleProxyStreaming(req *http.Request, ctx *goproxy.ProxyCtx) (rreq *http.Request, rresp *http.Response) {
	if rq, rs := proxyAuth(req, ctx); rs != nil {
		return rq, rs
	}

	if req.URL.Path != "/1.1/user.json" {
		return req, newResponse(req, http.StatusNotFound)
	}

	return req, handleStreaming(req, false)
}

func handleProxyApi(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	if rq, rs := proxyAuth(req, ctx); rs != nil {
		return rq, rs
	}

	return req, handleApi(req, false)
}
