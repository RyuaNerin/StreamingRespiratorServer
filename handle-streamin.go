package main

import (
	"io"
	"net/http"
)

var (
	StreamingResponseBadRequest     = []byte("잘못된 요청입니다..")
	StreamingResponseInvalidSession = []byte("세션이 만료되었습니다. 설정 파일을 갱신해주세요.")
)

func handleStreaming(req *http.Request, isHttpServer bool) *http.Response {
	act, ok := getTwitterClient(req, isHttpServer)
	if !ok {
		return newResponseWithText(req, http.StatusUnauthorized, StreamingResponseBadRequest)
	}

	if !act.VerifyCredentials() {
		return newResponseWithText(req, http.StatusUnauthorized, StreamingResponseInvalidSession)
	}

	r, w := io.Pipe()

	go func() {
		act.AddConnectionAndWait(w)
		w.Close()
	}()

	resp := newResponse(req, http.StatusOK)
	resp.Body = r
	resp.Header = http.Header{
		"Transfer-Encoding": []string{"chunked"},
		"Content-type":      []string{"application/json; charset=utf-8"},
		"Connection":        []string{"close"},
	}
	resp.TransferEncoding = []string{"chunked"}

	return resp
}
