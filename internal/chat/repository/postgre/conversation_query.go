package postgre

import (
	"github.com/aarondl/sqlboiler/v4/queries/qm"

	"knowledge-srv/internal/chat/repository"
)

// buildListConversationsQuery - Build query for ListConversations
func (r *implRepository) buildListConversationsQuery(opt repository.ListConversationsOptions) []qm.QueryMod {
	mods := []qm.QueryMod{}

	// Required filters
	if opt.CampaignID != "" {
		mods = append(mods, qm.Where("campaign_id = ?", opt.CampaignID))
	}
	if opt.UserID != "" {
		mods = append(mods, qm.Where("user_id = ?", opt.UserID))
	}

	// Optional filters
	if opt.Status != "" {
		mods = append(mods, qm.Where("status = ?", opt.Status))
	}

	// Sorting: most recent activity first
	mods = append(mods, qm.OrderBy("last_message_at DESC NULLS LAST, created_at DESC"))

	// Pagination
	if opt.Limit > 0 {
		mods = append(mods, qm.Limit(opt.Limit))
	}
	if opt.Offset > 0 {
		mods = append(mods, qm.Offset(opt.Offset))
	}

	return mods
}

// buildListMessagesQuery - Build query for ListMessages
func (r *implRepository) buildListMessagesQuery(opt repository.ListMessagesOptions) []qm.QueryMod {
	mods := []qm.QueryMod{}

	// Required filter
	if opt.ConversationID != "" {
		mods = append(mods, qm.Where("conversation_id = ?", opt.ConversationID))
	}

	// Sorting
	if opt.OrderASC {
		mods = append(mods, qm.OrderBy("created_at ASC"))
	} else {
		mods = append(mods, qm.OrderBy("created_at DESC"))
	}

	// Safety limit
	if opt.Limit > 0 {
		mods = append(mods, qm.Limit(opt.Limit))
	}

	return mods
}
