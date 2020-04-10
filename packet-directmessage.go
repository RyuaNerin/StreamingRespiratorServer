package main

import "time"

type PacketDirectMessage struct {
	Item PacketDirectMessageItem `json:"direct_message"`
}

type PacketDirectMessageItem struct {
	Id                  uint64      `json:"id"`
	IdStr               string      `json:"id_str"`
	Text                string      `json:"text"`
	CreatedAt           time.Time   `json:"created_at"`
	Sender              TwitterUser `json:"sender"`
	SenderId            uint64      `json:"sender_id"`
	SenderScreenName    string      `json:"sender_screen_name"`
	Recipient           TwitterUser `json:"recipient"`
	RecipientId         uint64      `json:"recipient_id"`
	RecipientScreenName string      `json:"recipient_screen_name"`
}
