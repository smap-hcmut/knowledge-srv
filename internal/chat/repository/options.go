package repository

import "encoding/json"

type CreateConversationOptions struct {
	CampaignID string
	UserID     string
	Title      string
}

type ListConversationsOptions struct {
	CampaignID string
	UserID     string
	Status     string
	Limit      int
	Offset     int
}

type UpdateLastMessageOptions struct {
	ConversationID string
	MessageCount   int
}

type CreateMessageOptions struct {
	ConversationID string
	Role           string
	Content        string
	Citations      json.RawMessage
	SearchMetadata json.RawMessage
	Suggestions    json.RawMessage
	FiltersUsed    json.RawMessage
}

type ListMessagesOptions struct {
	ConversationID string
	Limit          int
	OrderASC       bool
}
