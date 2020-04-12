package main

import (
	"io"
	"sort"
)

func tlAboutMeGetUrl(cursor string) (method string, url string) {
	method = "GET"

	if cursor == "" {
		url = "https://api.twitter.com/1.1/activity/about_me.json?model_version=7&skip_aggregation=true&cards_platform=Web-13&include_entities=1&include_user_entities=1&include_cards=1&send_error_codes=1&tweet_mode=extended&include_ext_alt_text=true&include_reply_count=true&count=1"
	} else {
		url = "https://api.twitter.com/1.1/activity/about_me.json?model_version=7&skip_aggregation=true&cards_platform=Web-13&include_entities=1&include_user_entities=1&include_cards=1&send_error_codes=1&tweet_mode=extended&include_ext_alt_text=true&include_reply_count=true&count=200&since_id=" + cursor
	}

	return
}

func tlAboutMeMain(r io.Reader, isFirstRefresh bool) (cursor string, packetList []Packet, users map[uint64]TwitterUser) {
	var activityList []TwitterActivity
	if err := jsonTwitter.NewDecoder(r).Decode(&activityList); err != nil && err != io.EOF {
		logger.Printf("%+v\n", err)
		return
	}
	if len(activityList) == 0 {
		return
	}

	if !isFirstRefresh {
		users = make(map[uint64]TwitterUser, len(activityList))

		statusList := make(TwitterStatusList, 0, len(activityList))
		for _, activity := range activityList {
			for _, t := range activity.Target {
				t.AddUserToMap(users)
			}
			for _, t := range activity.Sources {
				t.AddUserToMap(users)
			}

			switch {
			case Config.Filter.MyRetweet && activity.Action == "retweet":
				fallthrough
			case Config.Filter.RetweetWithComment && activity.Action == "quote":
				fallthrough
			case activity.Action == "reply":
				statusList = append(statusList, activity.Target...)
			}
		}

		sort.Sort(&statusList)
		for _, status := range statusList {
			if p, ok := newPacket(&status); ok {
				packetList = append(packetList, p)
			}
		}
	}

	for _, activity := range activityList {
		if activity.MaxPosition > cursor {
			cursor = activity.MaxPosition
		}
	}

	return
}
