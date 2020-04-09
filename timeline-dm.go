package main

import (
	"io"
	"strconv"

	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cast"
)

func tlDMGetUrl(cursor string) (method string, url string) {
	method = "GET"

	if cursor == "" {
		url = "https://api.twitter.com/1.1/dm/user_updates.json?include_groups=true&ext=altText&cards_platform=Web-13&include_entities=1&include_user_entities=1&include_cards=1&send_error_codes=1&tweet_mode=extended&include_ext_alt_text=true&include_reply_count=true&cursor=" + cursor
	} else {
		url = "https://api.twitter.com/1.1/dm/user_updates.json?include_groups=true&ext=altText&cards_platform=Web-13&include_entities=1&include_user_entities=1&include_cards=1&send_error_codes=1&tweet_mode=extended&include_ext_alt_text=true&include_reply_count=true"
	}

	return
}

func tlDMMain(r io.Reader, isFirstRefresh bool) (cursor string, streamingStatuses TwitterStatusList, users map[uint64]TwitterUser) {
	var statusList []TwitterStatus
	if err := jsoniter.NewDecoder(r).Decode(&statusList); err != nil && err != io.EOF {
		return
	}
	if len(statusList) == 0 {
		return
	}

	if !isFirstRefresh {
		streamingStatuses = make([]TwitterStatus, 0, len(statusList))

		users = make(map[uint64]TwitterUser)
		for _, status := range statusList {
			streamingStatuses = append(streamingStatuses, status)

			status.AddUserToMap(users)
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

	return
}
