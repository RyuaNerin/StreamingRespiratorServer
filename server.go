package main

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const (
	PathSelf = "/userstream"
)

var (
	CurrentConnections int32 = 0
)

type streamingRespiratorServer struct {
	httpHandler http.Handler

	httpClient http.Client

	tlsConfig *tls.Config
}

func newStreamingRespiratorServer(server2 *http2.Server, tlsConfig *tls.Config) *streamingRespiratorServer {
	s := &streamingRespiratorServer{
		tlsConfig: tlsConfig,
		httpClient: http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost:   32,
				Proxy:                 proxy,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 30 * time.Second,
				IdleConnTimeout:       30 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
			},
		},
	}

	mux := http.NewServeMux()
	mux.Handle("/api.twitter.com/", http.StripPrefix("/api.twitter.com/", http.HandlerFunc(s.handleApi)))
	mux.HandleFunc("/userstream.twitter.com/1.1/user.json", s.handleStreaming)
	mux.HandleFunc(PathSelf, s.handleStreaming)

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	s.httpHandler = h2c.NewHandler(mux, server2)

	return s
}

func (s *streamingRespiratorServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			sentry.CaptureException(err.(error))
			logger.Printf("%+v\n", errors.WithStack(err.(error)))
		}
	}()

	logger.Println("BEG |", atomic.AddInt32(&CurrentConnections, 1), "|", r.RemoteAddr, "|", r.Method, r.URL.String())
	defer func() {
		logger.Println("END |", atomic.AddInt32(&CurrentConnections, -1), "|", r.RemoteAddr, "|", r.Method, r.URL.String())
	}()

	if r.Method == "CONNECT" {
		s.handleProxyHttps(w, r)
		return
	}

	//////////////////////////////////////////////////
	// 프록시 영역 확인
	if r.URL.IsAbs() {
		s.handleProxyTunnel(w, r)
		return
	}

	//////////////////////////////////////////////////
	// 여기서부터는 프록시가 아님
	if !s.checkAuth(w, r) {
		return
	}

	s.httpHandler.ServeHTTP(w, r)
}
func (s *streamingRespiratorServer) handleProxyTunnel(w http.ResponseWriter, r *http.Request) {
	if !s.checkProxyAuth(w, r) {
		return
	}

	hi, ok := w.(http.Hijacker)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hi.Hijack()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	clientConnReader := bufio.NewReader(clientConn)

	var remoteConn net.Conn
	remoteHost := ""
	remoteConnReader := bufio.NewReader(nil)
	for {
		if r == nil {
			remoteConnReader.Reset(remoteConn)
			r, err = http.ReadRequest(clientConnReader)
			if err != nil && err != io.EOF {
				logger.Printf("%+v\n", err)
				sentry.CaptureException(err.(error))

				if remoteConn == nil {
					remoteConn.Close()
				}
				return
			}
		}

		if r == nil {
			remoteConn.Close()
			return
		}

		if s.isWebsocket(r) {
			s.handleProxyWebSocket(clientConn, clientConnReader, r, false)
			return
		}

		if host := s.getHost(r, 80); host != remoteHost {
			if remoteConn != nil {
				remoteConn.Close()
			}

			remoteConn, err = net.Dial("tcp", host)
			if err != nil {
				logger.Printf("%+v\n", err)
				sentry.CaptureException(err.(error))

				resp := http.Response{
					StatusCode: http.StatusBadGateway,
				}
				s.setResponse(&resp, r)
				err = resp.Write(remoteConn)
				if err != nil && err != io.EOF {
					logger.Printf("%+v\n", err)
					sentry.CaptureException(err.(error))
					remoteConn.Close()
					return
				}

				r = nil
				continue
			}
			remoteConnReader.Reset(remoteConn)
		}

		err = r.Write(remoteConn)
		if err != nil && err != io.EOF {
			resp := http.Response{
				StatusCode: http.StatusBadGateway,
			}
			s.setResponse(&resp, r)
			err = resp.Write(remoteConn)
			if err != nil && err != io.EOF {
				logger.Printf("%+v\n", err)
				sentry.CaptureException(err.(error))
				remoteConn.Close()
				return
			}
		}

		resp, err := http.ReadResponse(remoteConnReader, r)
		if err != nil {
			logger.Printf("%+v\n", err)
			sentry.CaptureException(err.(error))
			remoteConn.Close()
			return
		}

		err = resp.Write(clientConn)
		if err != nil && err != io.EOF {
			logger.Printf("%+v\n", err)
			remoteConn.Close()
			return
		}

		r = nil
	}
}
