package main

import (
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/elazarl/goproxy"
	"golang.org/x/net/http2"
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

		initProxy(proxy)

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
				server.TLSConfig = &tlsConfig
				http2.ConfigureServer(&server, &http2.Server{})
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
