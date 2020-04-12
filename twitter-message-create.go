package main

type TwitterMessageCreate struct {
	Event struct {
		Type string `json:"type"`

		MessageCreate struct {
			Target struct {
				RecipientId string `json:"recipient_id"`
			} `json:"target"`

			MessageData struct {
				Text string `json:"text"`
			} `json:"message_data"`
		} `json:"message_create"`
	} `json:"event"`
}
