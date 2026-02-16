package model

import (
	"encoding/json"
	"time"
)

// Message represents a single message in a conversation
type Message struct {
	ID             string
	ConversationID string
	Role           string // "user" | "assistant"
	Content        string
	Citations      json.RawMessage
	SearchMetadata json.RawMessage
	Suggestions    json.RawMessage
	FiltersUsed    json.RawMessage
	CreatedAt      time.Time
}
