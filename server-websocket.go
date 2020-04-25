package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
)

func (s *streamingRespiratorServer) isWebsocket(r *http.Request) bool {
	return r.Header.Get("Connection") == "upgrade" && r.Header.Get("Upgrade") == "websocket"
}

func (s *streamingRespiratorServer) handleProxyWebSocket(client io.ReadWriter, clientReader *bufio.Reader, r *http.Request, useTls bool) {
	if useTls {
		r.URL.Scheme = "wsw"
	} else {
		r.URL.Scheme = "ws"
	}

	var target io.ReadWriteCloser
	var err error

	if useTls {
		host, _, _ := net.SplitHostPort(s.getHost(r, 80))
		target, err = net.Dial("tcp", host)
	} else {
		host, _, _ := net.SplitHostPort(s.getHost(r, 443))
		target, err = tls.Dial("tcp", host, s.tlsConfig)
	}
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}
	defer target.Close()

	targerReader := bufio.NewReader(target)

	//////////////////////////////////////////////////
	err = r.Write(target)
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}

	resp, err := http.ReadResponse(targerReader, r)
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}

	err = resp.Write(target)
	if err != nil {
		fmt.Printf("%+v\n", err)
		return
	}

	//////////////////////////////////////////////////
	s.copy(client, clientReader, target)
}
