package main

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/spf13/cast"
)

func (s *streamingRespiratorServer) handleApi(w http.ResponseWriter, r *http.Request) {
	r.URL.Host = "api.twitter.com"
	r.Host = "api.twitter.com"

	act, ok := s.getTwitterClient(r)
	if ok {
		switch {
		//////////////////////////////////////////////////
		case strings.HasPrefix(r.URL.Path, "/1.1/statuses/destroy/"):
			fallthrough
		case strings.HasPrefix(r.URL.Path, "/1.1/statuses/unretweet/"):
			s.handleApiDestroyOrUnretweet(w, r, act)
			return

		//////////////////////////////////////////////////
		case strings.HasPrefix(r.URL.Path, "/1.1/statuses/retweet/"):
			s.handleApiRetweet(w, r, act)
			return

		//////////////////////////////////////////////////
		case strings.HasPrefix(r.URL.Path, "/1.1/statuses/update.json"):
			s.handleApiUpdate(w, r, act)
			return
		}
	}

	s.tunnelAndGetResponse(w, r, act, nil)
}

func (s *streamingRespiratorServer) handleApiDestroyOrUnretweet(w http.ResponseWriter, r *http.Request, act *Account) {
	var v TwitterStatus
	statusCode, vOk, ok := s.tunnelAndGetResponse(w, r, act, &v)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if vOk && statusCode == http.StatusOK {
		go act.sendStatusRemovedFromStatus(v)
	}
}

func (s *streamingRespiratorServer) handleApiRetweet(w http.ResponseWriter, r *http.Request, act *Account) {
	// 내 리트윗 다시 표시 기능을 끄면 별도 처리를 해줄 필요가 없음.
	if !Config.Filter.MyRetweet {
		_, _, _ = s.tunnelAndGetResponse(w, r, act, nil)
		return
	}

	var v TwitterStatus

	// 일단 api 호출하고...
	// 여기서 온 resp 는 항상 return 함.
	statusCode, vOk, ok := s.tunnelAndGetResponse(w, r, act, &v)
	if !ok {
		return
	}

	switch statusCode {
	case http.StatusOK: // full_text 있는지 확인
	case http.StatusNotFound: // 트윗이 삭제됨
		if id, ok := s.parseJsonId(r.URL.Path); ok {
			go act.SendStatusRemoved(id, 0)
		}
		return

	default:
		return
	}

	// full_text 가 있는지 확인
	if vOk {
		if _, ok := v["full_text"]; ok {
			if packet, ok := newPacket(&v); ok {
				go act.Send(packet)

				return
			}
		}
	}

	// full_text 를 가진 TwitterStatus 조회 후에 스트리밍쪽으로보내주는 작업
	go func() {
		req, _ := act.CreateRequest(
			context.Background(),
			"GET",
			"https://api.twitter.com/1.1/statuses/show.json?include_entities=1&tweet_mode=extended&id="+cast.ToString(v["id_str"]),
			nil,
		)
		res, err := act.httpClient.Do(req)
		if err != nil {
			return
		}
		defer res.Body.Close()

		// Serialize 없이 바로 전송
		packet, ok := newPacketFromReader(res.Body)
		if ok {
			act.Send(packet)
		}
	}()
}

var reHandleApiUpdateDM = regexp.MustCompile(`^d @?([A-Za-z0-9_]{3,15}) (.+)$`)

func (s *streamingRespiratorServer) handleApiUpdate(w http.ResponseWriter, r *http.Request, act *Account) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil && err != io.EOF {
		_, _, _ = s.tunnelAndGetResponse(w, r, act, nil)
		return
	}

	if r.PostForm == nil {
		_, _, _ = s.tunnelAndGetResponse(w, r, act, nil)
		return
	}

	if status := r.PostForm.Get("status"); status != "" {
		if m := reHandleApiUpdateDM.FindAllStringSubmatch(status, 1); len(m) > 0 {
			// Send DM
			userId, ok := act.GetUserId(ctx, m[0][1])
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			s.handleApiUpdateSendDm(w, r, act, userId, m[0][2])
			return
		}
	}

	statusCode, _, ok := s.tunnelAndGetResponse(w, r, act, nil)
	if !ok {
		return
	}

	// 트윗이 삭제된 경우 401 메시지가 발생한다.
	if statusCode == http.StatusUnauthorized {
		if inReplyToStatusId, err := cast.ToUint64E(r.PostForm.Get("in_reply_to_status_id")); err != nil {
			go act.SendStatusRemovedWithCheck(inReplyToStatusId)
		}
	}
}

func (s *streamingRespiratorServer) handleApiUpdateSendDm(w http.ResponseWriter, r *http.Request, act *Account, userId uint64, text string) {
	var dmData TwitterMessageCreate
	dmData.Event.Type = "message_create"
	dmData.Event.MessageCreate.Target.RecipientId = strconv.FormatUint(userId, 10)
	dmData.Event.MessageCreate.MessageData.Text = text

	buffRequest := BytesPool.Get().(*bytes.Buffer)
	defer BytesPool.Put(buffRequest)
	buffRequest.Reset()

	if err := jsonTwitter.NewEncoder(buffRequest).Encode(&dmData); err != nil && err != io.EOF {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	nreq, _ := act.CreateRequest(
		r.Context(),
		"POST",
		"https://api.twitter.com/1.1/direct_messages/events/new.json",
		buffRequest,
	)
	nreq.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := act.httpClient.Do(nreq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	buffResponse := BytesPool.Get().(*bytes.Buffer)
	defer BytesPool.Put(buffResponse)
	buffResponse.Reset()

	_, err = io.Copy(buffResponse, resp.Body)
	if err != nil && err != io.EOF {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if resp.StatusCode == http.StatusOK {
		s.writeResponse(w, resp, bytes.NewReader(nil), 0)
	} else {
		s.writeResponse(w, resp, bytes.NewReader(buffResponse.Bytes()), buffResponse.Len())
	}
}

// ok = false => internal server error
func (s *streamingRespiratorServer) tunnelAndGetResponse(w http.ResponseWriter, r *http.Request, act *Account, v interface{}) (statusCode int, vOk bool, ok bool) {
	req, _ := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), nil)

	if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
		buffRequest := BytesPool.Get().(*bytes.Buffer)
		BytesPool.Put(buffRequest)
		buffRequest.Reset()

		s.writeValuesWithEncoding(buffRequest, r.PostForm)

		req.Body = ioutil.NopCloser(buffRequest)
	}
	req.Header = make(http.Header)
	s.copyHeader(req.Header, r.Header)

	var resp *http.Response
	var err error
	if act != nil {
		resp, err = act.httpClient.Do(req)
	} else {
		resp, err = s.httpClient.Do(req)
	}
	if err != nil {
		logger.Printf("%+v\n", err)
		sentry.CaptureException(err.(error))
		w.WriteHeader(http.StatusInternalServerError)
		return 0, false, false
	}
	defer resp.Body.Close()

	buff := BytesPool.Get().(*bytes.Buffer)
	defer BytesPool.Put(buff)
	buff.Reset()

	_, err = io.Copy(buff, resp.Body)
	if err != nil && err != io.EOF {
		logger.Printf("%+v\n", err)
		sentry.CaptureException(err.(error))
		w.WriteHeader(http.StatusInternalServerError)
		return 0, false, false
	}

	if v == nil {
		if err = jsonTwitter.NewDecoder(bytes.NewReader(buff.Bytes())).Decode(&v); err == nil || err == io.EOF {
			vOk = true
		}
	}

	s.writeResponse(w, resp, bytes.NewReader(buff.Bytes()), buff.Len())

	return resp.StatusCode, vOk, true
}
