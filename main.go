package main

import (
	"crypto/tls"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/net/http2"
)

var (
	authId string
	authPw string

	verbose bool
	logger  *log.Logger
)

func main() {
	flag.BoolVar(&verbose, "verbose", false, "")

	argConfig := flag.String("config", "", "")
	flag.StringVar(&authId, "id", "", "")
	flag.StringVar(&authPw, "pw", "", "")

	argProxy := flag.String("proxy", "", "")

	argHttp := flag.String("http", "", "")
	argHttpPlain := flag.Bool("http-plain", false, "")
	argHttpCert := flag.String("http-cert", "", "")
	argHttpKey := flag.String("http-key", "", "")

	argUnix := flag.String("unix", "", "")
	argUnixPerm := flag.Int("unix-perm", 0700, "")

	argDebug := flag.Bool("debug", false, "")
	flag.Parse()

	if *argConfig == "" || authId == "" || authPw == "" {
		return
	}
	if *argProxy == "" && *argHttp == "" && *argUnix == "" {
		return
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
		server := http.Server{
			Handler:  newHttpMux(false),
			ErrorLog: logger,
		}

		l, err := net.Listen("tcp", *argHttp)
		if err != nil {
			panic(err)
		}

		if *argHttpPlain == false {
			tlsConfigHttp.BuildNameToCertificate()

			if *argHttpCert != "" || *argHttpKey != "" {
				clientCert, err := tls.LoadX509KeyPair(*argHttpCert, *argHttpKey)
				if err != nil {
					panic(err)
				}
				tlsConfigHttp.Certificates = append(tlsConfigHttp.Certificates, clientCert)
			}
			tlsConfigHttp.BuildNameToCertificate()

			server.TLSConfig = &tlsConfigHttp
			http2.ConfigureServer(&server, &http2.Server{})
			go server.ServeTLS(l, "", "")
		} else {
			go server.Serve(l)
		}

		defer l.Close()
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sig
}
