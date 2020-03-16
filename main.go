package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"

	"github.com/elazarl/goproxy"
)

var (
	authId string
	authPw string
)

func main() {
	argVerbose := flag.Bool("verbose", false, "verbose")

	argConfig := flag.String("config", "", "config")
	argBind := flag.String("bind", "", "bind")
	flag.StringVar(&authId, "id", "", "id")
	flag.StringVar(&authPw, "pw", "", "pw")
	flag.Parse()

	if *argConfig == "" || authId == "" || authPw == "" {
		return
	}

	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = *argVerbose
	proxy.CertStore = new(certStore)

	cond := proxy.OnRequest(goproxy.DstHostIs("userstream.twitter.com:443"))
	cond.HandleConnect(goproxy.AlwaysMitm)
	cond.DoFunc(handleStreaming)

	cond = proxy.OnRequest(goproxy.DstHostIs("api.twitter.com:443"))
	cond.HandleConnect(goproxy.AlwaysMitm)
	cond.DoFunc(handleApi)

	proxy.OnRequest().HandleConnect(
		goproxy.FuncHttpsHandler(
			func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
				return goproxy.OkConnect, host
			},
		),
	)

	proxy.NonproxyHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Host == "" {
			return
		}
		req.URL.Scheme = "http"
		req.URL.Host = req.Host
		proxy.ServeHTTP(w, req)
	})

	server := http.Server{
		Handler: proxy,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS11,
			NextProtos: []string{"http/1.1"},
		},
	}

	taddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", *argBind, Config.Proxy.Port))
	if err != nil {
		panic(err)
	}

	l, err := net.ListenTCP("tcp4", taddr)
	if err != nil {
		panic(err)
	}

	//tl := tls.NewListener(l, server.TLSConfig)

	_ = server.Serve(l)
}
