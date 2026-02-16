package model

import (
	"encoding/json"
	"time"

	"github.com/aarondl/null/v8"

	"knowledge-srv/internal/sqlboiler"
)

// Conversation represents a chat conversation session
type Conversation struct {
	ID            string
	CampaignID    string
	UserID        string
	Title         string
	Status        string // ACTIVE | ARCHIVED
	MessageCount  int
	LastMessageAt *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewConversationFromDB converts a SQLBoiler Conversation to model Conversation
func NewConversationFromDB(db *sqlboiler.Conversation) *Conversation {
	if db == nil {
		return nil
	}

	conv := &Conversation{
		ID:           db.ID,
		CampaignID:   db.CampaignID,
		UserID:       db.UserID,
		Title:        db.Title,
		Status:       db.Status,
		MessageCount: db.MessageCount,
	}

	// Handle nullable fields
	if db.LastMessageAt.Valid {
		conv.LastMessageAt = &db.LastMessageAt.Time
	}
	if db.CreatedAt.Valid {
		conv.CreatedAt = db.CreatedAt.Time
	}
	if db.UpdatedAt.Valid {
		conv.UpdatedAt = db.UpdatedAt.Time
	}

	return conv
}

// ToDBConversation converts model Conversation to SQLBoiler Conversation
func (c *Conversation) ToDBConversation() *sqlboiler.Conversation {
	db := &sqlboiler.Conversation{
		ID:           c.ID,
		CampaignID:   c.CampaignID,
		UserID:       c.UserID,
		Title:        c.Title,
		Status:       c.Status,
		MessageCount: c.MessageCount,
	}

	// Handle nullable fields
	if c.LastMessageAt != nil {
		db.LastMessageAt = null.TimeFrom(*c.LastMessageAt)
	}
	db.CreatedAt = null.TimeFrom(c.CreatedAt)
	db.UpdatedAt = null.TimeFrom(c.UpdatedAt)

	return db
}

// ConversationJSON is the JSON-serializable representation of Conversation
// Used only when the upstream caller needs JSON output
type ConversationJSON struct {
	ID            string     `json:"id"`
	CampaignID    string     `json:"campaign_id"`
	UserID        string     `json:"user_id"`
	Title         string     `json:"title"`
	Status        string     `json:"status"`
	MessageCount  int        `json:"message_count"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ToJSON converts a Conversation to its JSON representation
func (c *Conversation) ToJSON() ConversationJSON {
	return ConversationJSON{
		ID:            c.ID,
		CampaignID:    c.CampaignID,
		UserID:        c.UserID,
		Title:         c.Title,
		Status:        c.Status,
		MessageCount:  c.MessageCount,
		LastMessageAt: c.LastMessageAt,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

// Ensure the JSON import is always used
var _ = json.Marshal
