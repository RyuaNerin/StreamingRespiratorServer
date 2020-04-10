package main

import (
	"net/http"

	"github.com/elazarl/goproxy"
)

func CheckAuth(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, pw, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(http.StatusProxyAuthRequired)
			w.Header().Set("Proxy-Authenticate", "Basic realm=\"Access to Streamning-Respirator\"")
			return
		}
		if id != authId || pw != authPw {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		handler.ServeHTTP(w, r)
	})
}
func ProxyAuth(handler func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response)) func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		id, pw, ok := req.BasicAuth()
		if !ok {
			resp := &http.Response{
				Request:          req,
				TransferEncoding: req.TransferEncoding,
				StatusCode:       http.StatusProxyAuthRequired,
				Status:           http.StatusText(http.StatusProxyAuthRequired),
				Header: http.Header{
					"Proxy-Authenticate": []string{"Basic realm=\"Access to Streamning-Respirator\""},
				},
				ContentLength: 0,
			}
			return req, resp
		}
		if id != authId || pw != authPw {
			resp := &http.Response{
				Request:          req,
				TransferEncoding: req.TransferEncoding,
				StatusCode:       http.StatusUnauthorized,
				Status:           http.StatusText(http.StatusUnauthorized),
				Header: http.Header{
					"Proxy-Authenticate": []string{"Basic realm=\"Access to Streamning-Respirator\""},
				},
				ContentLength: 0,
			}
			return req, resp
		}

		return handler(req, ctx)
	}
}
