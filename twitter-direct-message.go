package main

type TwitterDirectMessage struct {
	UserInbox  *TwitterDirectMessageUserItem `json:"user_inbox,omitempty"`
	UserEvents *TwitterDirectMessageUserItem `json:"user_events,omitempty"`
}

type TwitterDirectMessageUserItem struct {
	Cursor  string                 `json:"cursor"`
	Users   map[string]TwitterUser `json:"users"`
	Entries []*struct {
		Message *struct {
			Data struct {
				Id          string `json:"id"`
				Time        string `json:"time"` // Milliseconds
				RecipiendId string `json:"recipient_id"`
				SenderId    string `json:"sender_id"`
				Text        string `json:"text"`
			} `json:"message_data"`
		} `json:"message"`
	} `json:"entries"`
}
