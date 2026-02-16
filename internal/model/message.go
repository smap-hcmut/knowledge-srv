package model

import (
	"encoding/json"
	"time"

	"github.com/aarondl/null/v8"

	"knowledge-srv/internal/sqlboiler"
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

// NewMessageFromDB converts a SQLBoiler Message to model Message
func NewMessageFromDB(db *sqlboiler.Message) *Message {
	if db == nil {
		return nil
	}

	msg := &Message{
		ID:             db.ID,
		ConversationID: db.ConversationID,
		Role:           db.Role,
		Content:        db.Content,
	}

	// Handle nullable JSON fields
	if db.Citations.Valid {
		msg.Citations = json.RawMessage(db.Citations.JSON)
	}
	if db.SearchMetadata.Valid {
		msg.SearchMetadata = json.RawMessage(db.SearchMetadata.JSON)
	}
	if db.Suggestions.Valid {
		msg.Suggestions = json.RawMessage(db.Suggestions.JSON)
	}
	if db.FiltersUsed.Valid {
		msg.FiltersUsed = json.RawMessage(db.FiltersUsed.JSON)
	}
	if db.CreatedAt.Valid {
		msg.CreatedAt = db.CreatedAt.Time
	}

	return msg
}

// ToDBMessage converts model Message to SQLBoiler Message
func (m *Message) ToDBMessage() *sqlboiler.Message {
	db := &sqlboiler.Message{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		Role:           m.Role,
		Content:        m.Content,
	}

	// Handle nullable JSON fields
	if len(m.Citations) > 0 && string(m.Citations) != "null" {
		db.Citations = null.JSONFrom(m.Citations)
	}
	if len(m.SearchMetadata) > 0 && string(m.SearchMetadata) != "null" {
		db.SearchMetadata = null.JSONFrom(m.SearchMetadata)
	}
	if len(m.Suggestions) > 0 && string(m.Suggestions) != "null" {
		db.Suggestions = null.JSONFrom(m.Suggestions)
	}
	if len(m.FiltersUsed) > 0 && string(m.FiltersUsed) != "null" {
		db.FiltersUsed = null.JSONFrom(m.FiltersUsed)
	}
	db.CreatedAt = null.TimeFrom(m.CreatedAt)

	return db
}
