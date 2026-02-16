package repository

import "encoding/json"

// CreateConversationOptions - Options cho CreateConversation
type CreateConversationOptions struct {
	CampaignID string
	UserID     string
	Title      string
}

// ListConversationsOptions - Options cho ListConversations
type ListConversationsOptions struct {
	CampaignID string
	UserID     string
	Status     string // optional filter
	Limit      int
	Offset     int
}

// UpdateLastMessageOptions - Options cho UpdateConversationLastMessage
type UpdateLastMessageOptions struct {
	ConversationID string
	MessageCount   int
}

// CreateMessageOptions - Options cho CreateMessage
type CreateMessageOptions struct {
	ConversationID string
	Role           string // "user" | "assistant"
	Content        string
	Citations      json.RawMessage
	SearchMetadata json.RawMessage
	Suggestions    json.RawMessage
	FiltersUsed    json.RawMessage
}

// ListMessagesOptions - Options cho ListMessages
type ListMessagesOptions struct {
	ConversationID string
	Limit          int  // max messages to load (default 20)
	OrderASC       bool // true = oldest first
}
