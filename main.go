package main

import (
	"crypto/tls"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var (
	authId string
	authPw string

	verbose bool
	logger  *log.Logger

	proxy func(*http.Request) (*url.URL, error)
)

func main() {
	flag.BoolVar(&verbose, "verbose", false, "")

	argConfig := flag.String("config", "", "")
	flag.StringVar(&authId, "id", "", "")
	flag.StringVar(&authPw, "pw", "", "")

	argProxy := flag.String("proxy", "", "")

	argHttp := flag.String("http", "", "")
	//argHttpPlain := flag.Bool("http-plain", false, "")
	argHttpCert := flag.String("http-cert", "", "")
	argHttpKey := flag.String("http-key", "", "")

	argUnix := flag.String("unix", "", "")
	argUnixPerm := flag.Int("unix-perm", 0700, "")

	argDebug := flag.Bool("debug", false, "")

	argClientHttpProxy := flag.String("client-http-proxy", "", "")
	flag.Parse()

	if *argConfig == "" || authId == "" || authPw == "" {
		return
	}
	if *argProxy == "" && *argHttp == "" && *argUnix == "" {
		return
	}

	if *argClientHttpProxy != "" {
		u, _ := url.Parse("http://" + *argClientHttpProxy)

		if sock, err := net.DialTimeout("tcp", *argClientHttpProxy, time.Second); err == nil {
			sock.Close()
			proxy = http.ProxyURL(u)
		}
	}

	if *argDebug {
		server := http.Server{
			Addr:    "127.0.0.1:12233",
			Handler: http.DefaultServeMux,
		}
		go server.ListenAndServe()
	}

	loadConfig(*argConfig)

	if verbose {
		logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	} else {
		logger = log.New(ioutil.Discard, "", 0)
		log.SetOutput(ioutil.Discard)
	}

	if *argProxy != "" {
		proxy := newProxyServer()

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

	if *argUnix != "" {
		server := http.Server{
			Handler:  newHttpMux(false),
			ErrorLog: logger,
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
		if *argHttpCert != "" || *argHttpKey != "" {
			clientCert, err := tls.LoadX509KeyPair(*argHttpCert, *argHttpKey)
			if err != nil {
				panic(err)
			}
			tlsConfigHttp.Certificates = append(tlsConfigHttp.Certificates, clientCert)
		}
		tlsConfigHttp.BuildNameToCertificate()

		var server2 http2.Server
		server := http.Server{
			Handler:   newHttpMux(true),
			ErrorLog:  logger,
			TLSConfig: &tlsConfigHttp,
		}
		http2.ConfigureServer(&server, &server2)
		server.Handler = h2c.NewHandler(server.Handler, &server2)

		l, err := net.Listen("tcp", *argHttp)
		if err != nil {
			panic(err)
		}
		l = newHyperListner(l, server.TLSConfig)

		go server.Serve(l)

		defer l.Close()
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sig
}
