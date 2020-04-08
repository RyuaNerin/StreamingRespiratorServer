package main

import "time"

type PacketEvent struct {
	Event     string      `json:"event"`
	CreatedAt time.Time   `json:"created_at"`
	Source    interface{} `json:"source"`
	Target    interface{} `json:"target"`
}
