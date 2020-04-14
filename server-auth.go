package main

import (
	"encoding/base64"
	"net/http"
	"strings"
)

const (
	proxyAuthorizationHeader = "Proxy-Authorization"
	authenticateHeaderValue  = "Basic realm=\"Access to Streaming-Respirator\""
)

func (s *streamingRespiratorServer) checkAuth(w http.ResponseWriter, req *http.Request) (ok bool) {
	id, pw, ok := s.splitProxyAuthorizationHeader(req, "Authorization")
	if !ok || id != authId || pw != authPw {
		id, pw, ok = req.BasicAuth()
		if !ok || id != authId || pw != authPw {
			w.Header().Set("WWW-Authenticate", authenticateHeaderValue)
			w.WriteHeader(http.StatusUnauthorized)
			return false
		}
	}

	return true
}

func (s *streamingRespiratorServer) checkProxyAuth(w http.ResponseWriter, r *http.Request) bool {
	statusCode := http.StatusOK
	id, pw, ok := s.splitProxyAuthorizationHeader(r, proxyAuthorizationHeader)
	if !ok {
		statusCode = http.StatusProxyAuthRequired
	} else if id != authId || pw != authPw {
		statusCode = http.StatusUnauthorized
	}

	if statusCode != http.StatusOK {
		w.Header().Set("Proxy-Authenticate", authenticateHeaderValue)
		w.Header().Set("Proxy-Connection", "close")
		w.WriteHeader(statusCode)

		return false
	}

	return true
}

func (s *streamingRespiratorServer) splitProxyAuthorizationHeader(r *http.Request, headerName string) (id string, pw string, ok bool) {
	authheader := strings.SplitN(r.Header.Get(proxyAuthorizationHeader), " ", 2)
	r.Header.Del(proxyAuthorizationHeader)
	if len(authheader) != 2 || authheader[0] != "Basic" {
		return
	}
	userpassraw, err := base64.StdEncoding.DecodeString(authheader[1])
	if err != nil {
		return
	}
	userpass := strings.SplitN(string(userpassraw), ":", 2)
	if len(userpass) != 2 {
		return
	}
	return userpass[0], userpass[1], true
}
