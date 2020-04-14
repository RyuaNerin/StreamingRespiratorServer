package main

import (
	"crypto/tls"
	"net"
	"net/http"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const PathSelf = "/userstream"

type streamingRespiratorServer struct {
	httpHandler http.Handler

	httpClient http.Client

	tlsConfig *tls.Config
}

func newStreamingRespiratorServer(server2 *http2.Server, tlsConfig *tls.Config) *streamingRespiratorServer {
	s := &streamingRespiratorServer{
		httpClient: http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 64,
				Proxy:               proxy,
			},
		},
		tlsConfig: tlsConfig,
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
			logger.Printf("%+v\n", err)
		}
	}()

	logger.Println("BEG |", r.RemoteAddr, "|", r.Method, r.URL.String())
	defer logger.Println("END |", r.RemoteAddr, "|", r.Method, r.URL.String())

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

	remoteConn, err := net.Dial("tcp", s.getHost(r, 80))
	if err != nil {
		logger.Printf("%+v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer remoteConn.Close()

	hj, ok := w.(http.Hijacker)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hj.Hijack()
	if err != nil {
		logger.Printf("%+v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//////////////////////////////////////////////////
	// Client Request -> remote
	if err = r.Write(remoteConn); err != nil {
		logger.Printf("%+v\n", err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	/**
	// remote Respose -> Server -> Client
	resp, err := http.ReadResponse(bufio.NewReader(remoteConn), r)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	err = resp.Write(clientConn)
	if err != nil {
		return
	}
	*/

	//////////////////////////////////////////////////
	// Copy Both
	s.copy(clientConn, remoteConn)
}
