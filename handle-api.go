package main

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cast"
)

func handleApi(req *http.Request, isHttpServer bool) *http.Response {
	act, ok := getTwitterClient(req, false)
	if ok {
		switch {
		//////////////////////////////////////////////////
		case strings.HasPrefix(req.URL.Path, "/1.1/statuses/destroy/"):
			fallthrough
		case strings.HasPrefix(req.URL.Path, "/1.1/statuses/unretweet/"):
			return handleApiDestroyOrUnretweet(act, req)

		//////////////////////////////////////////////////
		case strings.HasPrefix(req.URL.Path, "/1.1/statuses/retweet/"):
			return handleApiRetweet(act, req)

		//////////////////////////////////////////////////
		case strings.HasPrefix(req.URL.Path, "/1.1/statuses/update.json"):
			return handleApiUpdate(act, req)
		}
	}

	//////////////////////////////////////////////////
	// Tunnel
	if !isHttpServer {
		return nil
	}

	resp, err := act.httpClient.Do(req)
	if err != nil {
		return newResponse(req, http.StatusInternalServerError)
	}
	defer resp.Body.Close()

	buff := BytesPool.Get().(*bytes.Buffer)
	buff.Reset()

	_, err = io.Copy(buff, resp.Body)
	if err != nil && err != io.EOF {
		BytesPool.Put(buff)
		return newResponse(req, http.StatusInternalServerError)
	}

	return newResponseFromResponse(req, resp, buff)
}

func sendStatusRemovedFromStatus(act *Account, v TwitterStatus) {
	id, err := cast.ToUint64E(v["id"])
	if err == nil {
		return
	}
	user, err := cast.ToStringMapE(v["user"])
	if err != nil {
		return
	}
	userId, err := cast.ToUint64E(user["id"])
	if err != nil {
		return
	}

	sendStatusRemoved(act, id, userId)
}
func sendStatusRemoved(act *Account, id uint64, userId uint64) {
	var packetJson PacketDelete
	packetJson.Delete.Status = PacketDeleteStatus{
		Id:        id,
		IdStr:     strconv.FormatUint(id, 10),
		UserId:    userId,
		UserIdStr: strconv.FormatUint(userId, 10),
	}

	if packet, ok := newPacket(&packetJson); ok {
		act.Send(packet)
	}
}

func sendStatusRemovedWithCheck(act *Account, id uint64) {
	req, _ := act.CreateRequest(
		"GET",
		"https://api.twitter.com/1.1/statuses/show.json?id="+strconv.FormatUint(id, 10),
		nil,
	)

	res, err := act.httpClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	var v struct {
		Id uint64 `json:"id"`
	}
	if err := jsoniter.NewDecoder(res.Body).Decode(&v); err != nil && err != io.EOF {
		return
	}

	sendStatusRemoved(act, v.Id, 0)
}

func handleApiDestroyOrUnretweet(act *Account, req *http.Request) *http.Response {
	var v TwitterStatus
	resp, vOk, ok := tunnelAndGetResponse(act, req, &v)
	if !ok {
		return newResponse(req, http.StatusInternalServerError)
	}

	if vOk && resp.StatusCode == http.StatusOK {
		go sendStatusRemovedFromStatus(act, v)
	}

	return resp
}

func handleApiRetweet(act *Account, req *http.Request) *http.Response {
	// 내 리트윗 다시 표시 기능을 끄면 별도 처리를 해줄 필요가 없음.
	if !Config.Filter.MyRetweet {
		resp, _, _ := tunnelAndGetResponse(act, req, nil)
		return resp
	}

	var v TwitterStatus

	// 일단 api 호출하고...
	// 여기서 온 resp 는 항상 return 함.
	resp, vOk, ok := tunnelAndGetResponse(act, req, &v)
	if !ok {
		return resp
	}

	switch resp.StatusCode {
	case http.StatusOK: // full_text 있는지 확인
	case http.StatusNotFound: // 트윗이 삭제됨
		if id, ok := parseJsonId(req.URL.Path); ok {
			go sendStatusRemoved(act, id, 0)
		}
		return resp
	default:
		return resp
	}

	// full_text 가 있는지 확인
	if vOk {
		if _, ok := v["full_text"]; ok {
			if packet, ok := newPacket(&v); ok {
				go act.Send(packet)
				return resp
			}
		}
	}

	// full_text 를 가진 TwitterStatus 조회 후에 스트리밍쪽으로보내주는 작업
	go func() {
		req, _ := act.CreateRequest(
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

	return resp
}

var reHandleApiUpdateDM = regexp.MustCompile(`^d @?([A-Za-z0-9_]{3,15}) (.+)$`)

func handleApiUpdate(act *Account, req *http.Request) *http.Response {
	if err := req.ParseForm(); err != nil && err != io.EOF {
		resp, _, _ := tunnelAndGetResponse(act, req, nil)
		return resp
	}

	if req.PostForm != nil {
		resp, _, _ := tunnelAndGetResponse(act, req, nil)
		return resp
	}

	if status := req.PostForm.Get("status"); status != "" {
		if m := reHandleApiUpdateDM.FindAllStringSubmatch(status, 1); len(m) > 0 {
			// Send DM
			userId, ok := getUserId(act, m[0][1])
			if !ok {
				return newResponse(req, http.StatusNotFound)
			}

			return sendDirectMessage(act, req, userId, m[0][2])
		}
	}

	resp, _, ok := tunnelAndGetResponse(act, req, nil)
	if !ok {
		return resp
	}

	// 트윗이 삭제된 경우 401 메시지가 발생한다.
	if resp.StatusCode == http.StatusUnauthorized {
		if inReplyToStatusId, err := cast.ToUint64E(req.PostForm.Get("in_reply_to_status_id")); err != nil {
			go sendStatusRemovedWithCheck(act, inReplyToStatusId)
		}
	}

	return resp
}

func getUserId(act *Account, screenName string) (userId uint64, ok bool) {
	userId, _, ok = act.GetUserCache(0, screenName)
	if ok {
		return userId, true
	}

	req, _ := act.CreateRequest(
		"GET",
		"https://api.twitter.com/1.1/users/show.json?screen_name="+url.QueryEscape(screenName),
		nil,
	)
	resp, err := act.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var tu struct {
		Id uint64 `json:"id"`
	}
	if err := jsoniter.NewDecoder(resp.Body).Decode(&tu); err != nil && err != io.EOF {
		return
	}

	return tu.Id, true
}

func sendDirectMessage(act *Account, req *http.Request, userId uint64, screenName string) *http.Response {
	var dmData TwitterMessageCreate
	dmData.Event.Type = "message_create"
	dmData.Event.MessageCreate.Target.RecipientId = strconv.FormatUint(userId, 10)
	dmData.Event.MessageCreate.MessageData.Text = screenName

	buffRequest := BytesPool.Get().(*bytes.Buffer)
	defer BytesPool.Put(buffRequest)
	buffRequest.Reset()

	if err := jsoniter.NewEncoder(buffRequest).Encode(&dmData); err != nil && err != io.EOF {
		return newResponse(req, http.StatusInternalServerError)
	}

	nreq, _ := act.CreateRequest(
		"POST",
		"https://api.twitter.com/1.1/direct_messages/events/new.json",
		buffRequest,
	)
	nreq.Header = http.Header{
		"Content-Type": []string{"application/json; charset=utf-8"},
	}

	resp, err := act.httpClient.Do(nreq)
	if err != nil {
		return newResponse(req, http.StatusInternalServerError)
	}
	defer resp.Body.Close()

	buffResponse := BytesPool.Get().(*bytes.Buffer)
	buffResponse.Reset()

	_, err = io.Copy(buffResponse, resp.Body)
	if err != nil && err != io.EOF {
		BytesPool.Put(buffResponse)
		return newResponse(req, http.StatusInternalServerError)
	}

	if resp.StatusCode == http.StatusOK {
		return newResponse(req, resp.StatusCode)
	} else {
		return newResponseFromResponse(req, resp, buffResponse)
	}
}

// ok = false => internal server error
func tunnelAndGetResponse(act *Account, req *http.Request, v interface{}) (resp *http.Response, vOk bool, ok bool) {
	res, err := act.httpClient.Do(req)
	if err != nil {
		return newResponse(req, http.StatusInternalServerError), false, false
	}
	defer res.Body.Close()

	buff := BytesPool.Get().(*bytes.Buffer)
	buff.Reset()

	_, err = io.Copy(buff, res.Body)
	if err != nil && err != io.EOF {
		BytesPool.Put(buff)
		return newResponse(req, http.StatusInternalServerError), false, false
	}

	resp = newResponseFromResponse(req, resp, buff)

	if v == nil {
		if err = jsoniter.NewDecoder(bytes.NewReader(buff.Bytes())).Decode(&v); err == nil || err == io.EOF {
			vOk = true
		}
	}

	return resp, vOk, true
}
