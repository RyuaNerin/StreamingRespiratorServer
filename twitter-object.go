package main

import (
	"github.com/spf13/cast"
)

type TwitterStatusList []TwitterStatus
type TwitterStatus map[string]interface{}

type TwitterUser map[string]interface{}

func (ts *TwitterStatusList) Len() int {
	return len(*ts)
}
func (ts *TwitterStatusList) Less(i, k int) bool {
	return cast.ToUint64((*ts)[i]["id"]) < cast.ToUint64((*ts)[k]["id"])
}
func (ts *TwitterStatusList) Swap(i, k int) {
	(*ts)[i], (*ts)[k] = (*ts)[k], (*ts)[i]
}

func (ts TwitterStatus) AddUserToMap(users map[uint64]TwitterUser) {
	if user, err := cast.ToStringMapE(ts["user"]); err != nil {
		if id, err := cast.ToUint64E(user["id"]); err == nil {
			users[id] = user
		}
	}

	if retweetedStatus, err := cast.ToStringMapE(ts["retweeted_status"]); err == nil {
		TwitterStatus(retweetedStatus).AddUserToMap(users)
	}

	if quotedStatus, err := cast.ToStringMapE(ts["quoted_status"]); err == nil {
		TwitterStatus(quotedStatus).AddUserToMap(users)
	}
}

func (tu TwitterUser) AddUserToMap(users map[uint64]TwitterUser) {
	if id, err := cast.ToUint64E(tu["id"]); err == nil {
		users[id] = tu
	}
}
