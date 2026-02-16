package postgre

import (
	"time"

	"github.com/aarondl/null/v8"

	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/sqlboiler"
)

// buildCreateConversation - Build sqlboiler entity from CreateConversationOptions
func buildCreateConversation(opt repository.CreateConversationOptions) *sqlboiler.Conversation {
	now := time.Now()

	return &sqlboiler.Conversation{
		CampaignID:   opt.CampaignID,
		UserID:       opt.UserID,
		Title:        opt.Title,
		Status:       "ACTIVE",
		MessageCount: 0,
		CreatedAt:    null.TimeFrom(now),
		UpdatedAt:    null.TimeFrom(now),
	}
}
