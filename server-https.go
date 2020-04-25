package main

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
)

const (
	MaxRequestBody = 10 * 1024 * 1024 // 10 MiB
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

	// 웹소켓은 MITM 안함
	host, _, _ := net.SplitHostPort(s.getHost(r, 443))
	switch host {
	case "api.twitter.com":
		fallthrough
	case "userstream.twitter.com":
		s.handleProxyHttpsMitm(clientConn, r)
		return
	}
	s.handleProxyHttpsTunnel(clientConn, r)
}

func (s *streamingRespiratorServer) handleProxyHttpsMitm(clientConn net.Conn, r *http.Request) {
	var err error

	if strings.ToLower(r.Header.Get("Proxy-Connection")) == "keep-alive" ||
		strings.ToLower(r.Header.Get("Connection")) == "keep-alive" {
		_, err = clientConn.Write(connectionEstablishedKA)
	} else {
		_, err = clientConn.Write(connectionEstablished)
	}
	if err != nil {
		return
	}

	clientConnTls := tls.Server(clientConn, s.tlsConfig)
	if err = clientConnTls.Handshake(); err != nil {
		logger.Printf("%+v\n", err)
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
		r.URL.Scheme = "https"

		//////////////////////////////////////////////////
		if s.isWebsocket(r) {
			s.handleProxyWebSocket(clientConnTls, clientConnTlsReader, r, true)
			return
		}

		//////////////////////////////////////////////////
		// 너무 큰 Request 는 스킵한다.

		if contentLength, err := strconv.ParseInt(r.Header.Get("Content-Type"), 10, 64); err == nil {
			if contentLength > MaxRequestBody {
				resp := http.Response{
					StatusCode: http.StatusRequestEntityTooLarge,
					Body:       nil,
				}

				err = resp.Write(clientConnTls)
				if err != nil && err != io.EOF {
					logger.Printf("%+v\n", err)
					return
				}
				continue
			}
		}

		rbr, rbw := io.Pipe()

		respWriter := ProxyResponseWriter{
			w:          rbw,
			header:     make(http.Header),
			statusCode: make(chan int),
		}

		if r.Host == "api.twitter.com" {
			go func() {
				s.handleApi(&respWriter, r)
				rbw.Close()
			}()
		} else {
			go func() {
				s.handleStreaming(&respWriter, r)
				rbw.Close()
			}()
		}
		respStatusCode := <-respWriter.statusCode
		resp := http.Response{
			StatusCode:    respStatusCode,
			Header:        respWriter.header,
			ContentLength: respWriter.contentLength,
			Body:          rbr,

			/**
			Body: &debugStream{
				name: r.URL.String(),
				ioi:  rbr,
			},
			*/
		}
		s.setResponse(&resp, r)

		err = resp.Write(clientConnTls)

		rbr.Close()
		rbw.Close()

		if err != nil && err != io.EOF {
			logger.Printf("%+v\n", err)
			return
		}
	}
}
func (s *streamingRespiratorServer) handleProxyHttpsTunnel(clientConn net.Conn, r *http.Request) {
	remoteConn, err := net.Dial("tcp", s.getHost(r, 443))
	if err != nil {
		logger.Printf("%+v\n", err)
		clientConn.Write(connectionFailed)
		return
	}
	defer remoteConn.Close()

	switch {
	case strings.ToLower(r.Header.Get("Proxy-Connection")) == "keep-alive":
		fallthrough
	case strings.ToLower(r.Header.Get("Connection")) == "keep-alive":
		_, err = clientConn.Write(connectionEstablishedKA)

	default:
		_, err = clientConn.Write(connectionEstablished)
	}
	if err != nil {
		logger.Printf("%+v\n", err)
		return
	}

	s.copy(clientConn, bufio.NewReader(clientConn), remoteConn, r.Context())
}

type ProxyResponseWriter struct {
	statusCode    chan int
	header        http.Header
	contentLength int64

	w io.Writer
}

func (w *ProxyResponseWriter) Header() http.Header {
	return w.header
}
func (w *ProxyResponseWriter) Write(p []byte) (int, error) {
	logger.Println("ProxyResponseWriter write", len(p))
	return w.w.Write(p)
}
func (w *ProxyResponseWriter) WriteHeader(statusCode int) {
	logger.Println(w.header.Get("Content-Length"))

	if c, err := strconv.ParseInt(w.header.Get("Content-Length"), 10, 64); err == nil {
		w.contentLength = c
	}
	w.statusCode <- statusCode
}
