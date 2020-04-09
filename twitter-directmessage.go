package main


type DirectMessage struct {
	UserInbox DirectMessageUserItem `json:"user_inbox"`
	UserEvents DirectMessageUserItem `json:"user_events"`
}

type DirectMessageUserItem  struct {
	Conversations interface{} `json:"conversations"`
	Cursor string `json:"curosr"`
	Entries DirectMessageEntries `json:"entries"`
	Users map[string]TwitterUser `json:"users"`

}

[DebuggerDisplay("Item")]
internal class DirectMessage
{

	[DebuggerDisplay("{Message}")]
	public class Entry
	{
		[JsonProperty("message")]
		public Message Message { get; set; }
	}

	[DebuggerDisplay("{Data}")]
	public class Message
	{
		[JsonProperty("message_data")]
		public MessageData Data { get; set; }
	}

	[DebuggerDisplay("{Sender_Id} > {Recipiend_Id} : {Id} / {Text}")]
	public class MessageData
	{
		[JsonProperty("id")]
		public long Id { get; set; }

		[JsonProperty("time")]
		public long Time { get; set; }

		[JsonIgnore]
		public DateTime CreatedAt => new DateTime(1970, 1, 1, 0, 0, 0).AddMilliseconds(this.Time);

		[JsonProperty("recipient_id")]
		public string Recipiend_Id { get; set; }

		[JsonProperty("sender_id")]
		public string Sender_Id { get; set; }

		[JsonProperty("text")]
		public string Text { get; set; }
	}
}