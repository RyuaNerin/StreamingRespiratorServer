package main

import (
	"net/http"
)

var (
	StreamingResponseBadRequest     = []byte("잘못된 요청입니다..")
	StreamingResponseInvalidSession = []byte("세션이 만료되었습니다. 설정 파일을 갱신해주세요.")
)

func (s *streamingRespiratorServer) handleStreaming(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	act, ok := s.getTwitterClient(req)
	if !ok {
		s.writeBytes(w, http.StatusUnauthorized, StreamingResponseBadRequest)
		return
	}

	if !act.VerifyCredentials(ctx) {
		s.writeBytes(w, http.StatusUnauthorized, StreamingResponseInvalidSession)
		return
	}

	act.AddConnectionAndWait(w, ctx)
}
