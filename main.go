package main

import (
	"crypto/tls"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/elazarl/goproxy"
)

var (
	authId string
	authPw string

	verbose bool
)

func main() {
	flag.BoolVar(&verbose, "verbose", false, "verbose")

	argConfig := flag.String("config", "", "")
	flag.StringVar(&authId, "id", "", "")
	flag.StringVar(&authPw, "pw", "", "")

	argProxy := flag.String("proxy", "", "")

	argHttp := flag.String("http", "", "")
	argHttpPlain := flag.Bool("http-plain", false, "")

	argUnix := flag.String("unix", "", "")
	argUnixPerm := flag.Int("unix-perm", 0700, "")
	flag.Parse()

	if *argConfig == "" || authId == "" || authPw == "" {
		return
	}

	LoadConfig(*argConfig)

	if *argProxy != "" {
		proxy := goproxy.NewProxyHttpServer()
		proxy.Verbose = verbose
		proxy.CertStore = new(certStore)

		cond := proxy.OnRequest(goproxy.DstHostIs("userstream.twitter.com"))
		cond.HandleConnect(goproxy.AlwaysMitm)
		cond.DoFunc(handleStreaming)

		cond = proxy.OnRequest(goproxy.DstHostIs("api.twitter.com"))
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
			w.WriteHeader(http.StatusBadRequest)
		})

		server := http.Server{
			Handler: proxy,
		}

		l, err := net.Listen("tcp", *argProxy)
		if err != nil {
			panic(err)
		}

		go server.Serve(l)
		defer l.Close()
	}

	if *argUnix != "" || *argHttp != "" {
		var mux http.ServeMux

		if *argUnix != "" {
			server := http.Server{
				Handler: &mux,
			}

			l, err := net.Listen("unix", *argUnix)
			if err != nil {
				panic(err)
			}

			err = os.Chmod(*argUnix, os.FileMode(*argUnixPerm))
			if err != nil {
				panic(err)
			}

			go server.Serve(l)
			defer l.Close()
		}

		if *argHttp != "" {
			server := http.Server{
				Handler: &mux,
			}

			l, err := net.Listen("unix", *argUnix)
			if err != nil {
				panic(err)
			}

			err = os.Chmod(*argUnix, os.FileMode(*argUnixPerm))
			if err != nil {
				panic(err)
			}

			if *argHttpPlain == false {
				server.TLSConfig = &tls.Config{
					MinVersion:   tls.VersionTLS11,
					NextProtos:   []string{"http/1.1"},
					Certificates: []tls.Certificate{certClient},
				}

				go server.ServeTLS(l, "", "")
			} else {
				go server.Serve(l)
			}

			defer l.Close()
		}
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sig
}
