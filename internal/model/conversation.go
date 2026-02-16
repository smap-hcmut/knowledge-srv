package model

import "time"

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
