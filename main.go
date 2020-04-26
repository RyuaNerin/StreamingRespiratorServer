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

	argBind := flag.String("bind", "", "")
	argBindCert := flag.String("bind-cert", "", "")
	argBindKey := flag.String("bind-key", "", "")

	argUnix := flag.String("unix", "", "")
	argUnixPerm := flag.Int("unix-perm", 0700, "")

	argDebug := flag.Bool("debug", false, "")

	argProxy := flag.String("proxy", "", "")
	flag.Parse()

	if *argConfig == "" || authId == "" || authPw == "" {
		return
	}
	if *argBind == "" && *argUnix == "" {
		return
	}

	if *argProxy != "" {
		u, _ := url.Parse("http://" + *argProxy)

		if sock, err := net.DialTimeout("tcp", *argProxy, time.Second); err == nil {
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

	////////////////////////////////////////////////////////////////////////////////////////////////////
	if *argBindCert != "" || *argBindKey != "" {
		clientCert, err := tls.LoadX509KeyPair(*argBindCert, *argBindKey)
		if err != nil {
			panic(err)
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, clientCert)
	}
	tlsConfig.BuildNameToCertificate()

	server2 := http2.Server{}
	server := http.Server{
		Handler:           newStreamingRespiratorServer(&server2, &tlsConfig),
		ErrorLog:          logger,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 30 * time.Second,
	}
	http2.ConfigureServer(&server, &server2)

	////////////////////////////////////////////////////////////////////////////////////////////////////

	if *argUnix != "" {
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

	if *argBind != "" {
		l, err := net.Listen("tcp", *argBind)
		if err != nil {
			panic(err)
		}
		l = newHyperListner(l, &tlsConfig)

		go server.Serve(l)
		defer l.Close()
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sig
}
