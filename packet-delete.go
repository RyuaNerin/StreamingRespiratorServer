package main

type PacketDelete struct {
	Delete struct {
		Status PacketDeleteStatus `json:"status"`
	} `json:"delete"`
}
type PacketDeleteStatus struct {
	Id        uint64 `json:"id"`
	IdStr     string `json:"id_str"`
	UserId    uint64 `json:"user_id"`
	UserIdStr string `json:"user_id_str"`
}
