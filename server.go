package main

import (
	"bufio"
	"crypto/tls"
	"io"
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

	hi, ok := w.(http.Hijacker)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	clientConn, clientConnRW, err := hi.Hijack()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	clientConnReader := bufio.NewReader(clientConn)

	var conn net.Conn
	connHost := ""
	connReader := bufio.NewReader(nil)
	for {
		if r == nil {
			r, err = http.ReadRequest(clientConnReader)
			if err != nil && err != io.EOF {
				logger.Printf("%+v\n", err)

				if conn == nil {
					conn.Close()
				}
				return
			}
		}

		if host := s.getHost(r, 80); host != connHost {
			if conn != nil {
				conn.Close()
			}

			conn, err = net.Dial("tcp", host)
			if err != nil {
				logger.Printf("%+v\n", err)

				resp := http.Response{
					StatusCode: http.StatusBadGateway,
				}
				s.setResponse(&resp, r)
				err = resp.Write(conn)
				if err != nil && err != io.EOF {
					logger.Printf("%+v\n", err)
					conn.Close()
					return
				}

				r = nil
				continue
			}
			connReader.Reset(conn)
		}

		err = r.Write(conn)
		if err != nil && err != io.EOF {
			resp := http.Response{
				StatusCode: http.StatusBadGateway,
			}
			s.setResponse(&resp, r)
			err = resp.Write(conn)
			if err != nil && err != io.EOF {
				logger.Printf("%+v\n", err)
				conn.Close()
				return
			}
		}

		resp, err := http.ReadResponse(connReader, r)
		if err != nil {
			logger.Printf("%+v\n", err)
			conn.Close()
			return
		}

		err = resp.Write(clientConnRW.Writer)
		if err != nil && err != io.EOF {
			logger.Printf("%+v\n", err)
			conn.Close()
			return
		}

		err = clientConnRW.Writer.Flush()
		if err != nil && err != io.EOF {
			logger.Printf("%+v\n", err)
			conn.Close()
			return
		}

		r = nil
	}
}
