package main

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strings"
)

var (
	connectionEstablishedKA = []byte("HTTP/1.1 200 Connection Established\r\nConnection: keep-alive\r\nKeep-Alive: timeout=30\r\n\r\n")
	connectionEstablished   = []byte("HTTP/1.1 200 Connection Established\r\nConnection: close\r\n\r\n")
	connectionFailed        = []byte("HTTP/1.1 502 Connection Failed\r\nConnection: close\r\n\r\n")
)

func (s *streamingRespiratorServer) handleProxyHttps(w http.ResponseWriter, r *http.Request) {
	if !s.checkProxyAuth(w, r) {
		return
	}

	r.Body.Close()

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

	switch r.Host {
	case "api.twitter.com":
		fallthrough
	case "userstream.twitter.com":
		s.handleProxyHttpsMitm(clientConn, r)

	default:
		s.handleProxyHttpsTunnel(clientConn, r)
	}
}

func (s *streamingRespiratorServer) handleProxyHttpsMitm(clientConn net.Conn, r *http.Request) {
	var err error

	switch {
	case strings.ToLower(r.Header.Get("Proxy-Connection")) == "keep-alive":
	case strings.ToLower(r.Header.Get("Connection")) == "keep-alive":
		_, err = clientConn.Write(connectionEstablishedKA)

	default:
		_, err = clientConn.Write(connectionEstablished)
	}
	if err != nil {
		return
	}

	clientConnTls := tls.Server(clientConn, s.tlsConfig)
	if err = clientConnTls.Handshake(); err != nil {
		return
	}

	clientConnTlsReader := bufio.NewReader(clientConnTls)
	for {
		if _, err := clientConnTlsReader.Peek(1); err == io.EOF {
			return
		}

		r, err = http.ReadRequest(clientConnTlsReader)
		if err != nil && err != io.EOF {
			logger.Printf("%+v\n", err)
			return
		}

		rbr, rbw := io.Pipe()

		respWriter := ProxyResponseWriter{
			w:          rbw,
			header:     make(http.Header),
			statusCode: make(chan int),
		}

		if r.Host == "api.twitter.com" {
			go s.handleApi(&respWriter, r)
		} else {
			go s.handleStreaming(&respWriter, r)
		}
		resp := http.Response{
			StatusCode: <-respWriter.statusCode,
			Header:     respWriter.header,
			Body:       rbr,
		}
		s.setResponse(&resp, r)

		err = resp.Write(clientConnTls)
		if err != nil && err != io.EOF {
			logger.Printf("%+v\n", err)
		}
	}
}
func (s *streamingRespiratorServer) handleProxyHttpsTunnel(clientConn net.Conn, r *http.Request) {
	remoteConn, err := net.Dial("tcp", s.getHost(r, 443))
	if err != nil {
		clientConn.Write(connectionFailed)
		clientConn.Close()
		return
	}
	defer remoteConn.Close()

	switch {
	case strings.ToLower(r.Header.Get("Proxy-Connection")) == "keep-alive":
	case strings.ToLower(r.Header.Get("Connection")) == "keep-alive":
		_, err = clientConn.Write(connectionEstablishedKA)

	default:
		_, err = clientConn.Write(connectionEstablished)
	}
	if err != nil {
		return
	}

	s.copy(clientConn, remoteConn)
}

type ProxyResponseWriter struct {
	statusCode chan int
	header     http.Header

	w io.Writer
}

func (w *ProxyResponseWriter) Header() http.Header {
	return w.header
}
func (w *ProxyResponseWriter) Write(p []byte) (int, error) {
	return w.Write(p)
}
func (w *ProxyResponseWriter) WriteHeader(statusCode int) {
	w.statusCode <- statusCode
}
