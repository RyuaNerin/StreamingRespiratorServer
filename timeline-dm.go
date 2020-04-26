package main

import (
	"io"
	"sort"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/spf13/cast"
)

func tlDMGetUrl(cursor string) (method string, url string) {
	method = "GET"

	if cursor == "" {
		url = "https://api.twitter.com/1.1/dm/user_updates.json?include_groups=true&ext=altText&cards_platform=Web-13&include_entities=1&include_user_entities=1&include_cards=1&send_error_codes=1&tweet_mode=extended&include_ext_alt_text=true&include_reply_count=true"
	} else {
		url = "https://api.twitter.com/1.1/dm/user_updates.json?include_groups=true&ext=altText&cards_platform=Web-13&include_entities=1&include_user_entities=1&include_cards=1&send_error_codes=1&tweet_mode=extended&include_ext_alt_text=true&include_reply_count=true&cursor=" + cursor
	}

	return
}

func tlDMMain(r io.Reader, isFirstRefresh bool) (cursor string, packetList []Packet, users map[uint64]TwitterUser) {
	var directMessage TwitterDirectMessage
	if err := jsonTwitter.NewDecoder(r).Decode(&directMessage); err != nil && err != io.EOF {
		logger.Printf("%+v\n", err)
		sentry.CaptureException(err.(error))
		return
	}

	data := directMessage.UserEvents
	if data == nil {
		data = directMessage.UserInbox
	}
	if data == nil {
		return
	}

	if !isFirstRefresh {
		if len(data.Entries) > 0 {
			users = make(map[uint64]TwitterUser)

			packetJsonList := make([]PacketDirectMessage, len(data.Entries))

			for _, entry := range data.Entries {
				if entry.Message != nil {
					id, err := strconv.ParseUint(entry.Message.Data.Id, 10, 64)
					if err != nil {
						continue
					}

					t, err := strconv.ParseInt(entry.Message.Data.Time, 10, 64)
					if err != nil {
						continue
					}

					// ToPacket
					packetJson := PacketDirectMessage{
						Item: PacketDirectMessageItem{
							Id:        id,
							IdStr:     entry.Message.Data.Id,
							CreatedAt: time.Unix(t/1000, t%1000),
							Recipient: data.Users[entry.Message.Data.RecipiendId],
							Sender:    data.Users[entry.Message.Data.SenderId],
						},
					}
					if packetJson.Item.Recipient != nil {
						packetJson.Item.RecipientId = cast.ToUint64(packetJson.Item.Recipient["id"])
						packetJson.Item.RecipientScreenName = cast.ToString(packetJson.Item.Recipient["screen_name"])
					}
					if packetJson.Item.Sender != nil {
						packetJson.Item.SenderId = cast.ToUint64(packetJson.Item.Sender["id"])
						packetJson.Item.SenderScreenName = cast.ToString(packetJson.Item.Sender["screen_name"])
					}
					packetJsonList = append(packetJsonList, packetJson)
				}
			}

			for _, user := range data.Users {
				user.AddUserToMap(users)
			}

			sort.Slice(packetJsonList, func(i, k int) bool {
				return packetJsonList[i].Item.Id < packetJsonList[k].Item.Id
			})
			for _, packetJson := range packetJsonList {
				if packet, ok := newPacket(&packetJson); ok {
					packetList = append(packetList, packet)
				}
			}
		}
	}

	cursor = data.Cursor

	return
}
