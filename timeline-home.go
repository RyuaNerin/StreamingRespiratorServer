package main

import (
	"io"
	"strconv"

	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cast"
)

func tlHomeGetUrl(cursor string) (method string, url string) {
	method = "GET"

	if cursor == "" {
		url = "https://api.twitter.com/1.1/statuses/home_timeline.json?&include_my_retweet=1&cards_platform=Web-13&include_entities=1&include_user_entities=1&include_cards=1&send_error_codes=1&tweet_mode=extended&include_ext_alt_text=true&include_reply_count=true&count=200&since_id=" + cursor
	} else {
		url = "https://api.twitter.com/1.1/statuses/home_timeline.json?&include_my_retweet=1&cards_platform=Web-13&include_entities=1&include_user_entities=1&include_cards=1&send_error_codes=1&tweet_mode=extended&include_ext_alt_text=true&include_reply_count=true&count=1"
	}

	return
}

func tlHomeMain(r io.Reader, isFirstRefresh bool) (cursor string, streamingStatuses []TwitterStatus, users map[uint64]TwitterUser) {
	var statusList []map[string]interface{}

	if err := jsoniter.NewDecoder(r).Decode(&statusList); (err == nil || err != io.EOF) && len(statusList) > 0 {
		if !isFirstRefresh {
			streamingStatuses = make([]TwitterStatus, 0, len(statusList))

			for _, status := range statusList {
				streamingStatuses = append(streamingStatuses, status)
			}
		}

		var maxId uint64 = 0
		for _, t := range statusList {
			id := cast.ToUint64(t["id"])
			if maxId > id {
				maxId = id
			}
		}
		cursor = strconv.FormatUint(maxId, 10)
	}

	return
}
