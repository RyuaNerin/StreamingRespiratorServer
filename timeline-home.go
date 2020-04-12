package main

import (
	"io"
	"sort"
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

func tlHomeMain(r io.Reader, isFirstRefresh bool) (cursor string, packetList []Packet, users map[uint64]TwitterUser) {
	var statusList TwitterStatusList
	if err := jsoniter.NewDecoder(r).Decode(&statusList); err != nil && err != io.EOF {
		return
	}
	if len(statusList) == 0 {
		return
	}

	if !isFirstRefresh {
		users = make(map[uint64]TwitterUser)

		for _, status := range statusList {
			status.AddUserToMap(users)
		}

		sort.Sort(&statusList)
		for status := range statusList {
			if p, ok := newPacket(&status); ok {
				packetList = append(packetList, p)
			}
		}
	}

	var maxId uint64 = 0
	for _, t := range statusList {
		id := cast.ToUint64(t["id"])
		if id > maxId {
			maxId = id
		}
	}
	cursor = strconv.FormatUint(maxId, 10)

	return
}
