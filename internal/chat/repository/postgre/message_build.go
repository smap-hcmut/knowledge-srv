package postgre

import (
	"github.com/aarondl/null/v8"

	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/sqlboiler"
)

// buildCreateMessage - Build sqlboiler entity from CreateMessageOptions
func buildCreateMessage(opt repository.CreateMessageOptions) *sqlboiler.Message {
	dbMsg := &sqlboiler.Message{
		ConversationID: opt.ConversationID,
		Role:           opt.Role,
		Content:        opt.Content,
	}

	// Handle nullable JSON fields
	if len(opt.Citations) > 0 && string(opt.Citations) != "null" {
		dbMsg.Citations = null.JSONFrom(opt.Citations)
	}
	if len(opt.SearchMetadata) > 0 && string(opt.SearchMetadata) != "null" {
		dbMsg.SearchMetadata = null.JSONFrom(opt.SearchMetadata)
	}
	if len(opt.Suggestions) > 0 && string(opt.Suggestions) != "null" {
		dbMsg.Suggestions = null.JSONFrom(opt.Suggestions)
	}
	if len(opt.FiltersUsed) > 0 && string(opt.FiltersUsed) != "null" {
		dbMsg.FiltersUsed = null.JSONFrom(opt.FiltersUsed)
	}

	return dbMsg
}
