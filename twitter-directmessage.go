package main

type TwitterDirectMessage struct {
	UserInbox  *TwitterDirectMessageUserItem `json:"user_inbox"`
	UserEvents *TwitterDirectMessageUserItem `json:"user_events"`
}

type TwitterDirectMessageUserItem struct {
	Conversations interface{}            `json:"conversations"`
	Cursor        string                 `json:"curosr"`
	Users         map[string]TwitterUser `json:"users"`
	Entries       []struct {
		Message *TwitterDirectMessageMessage `json:"message"`
	} `json:"entries"`
}

type TwitterDirectMessageMessage struct {
	Data struct {
		Id          uint64 `json:"id"`
		Time        int64  `json:"time"` // Milliseconds
		RecipiendId string `json:"recipient_id"`
		SenderId    string `json:"sender_id"`
		Text        string `json:"text"`
	} `json:"message_data"`
}
