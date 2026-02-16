package chat

import "errors"

var (
	ErrConversationNotFound = errors.New("chat: conversation not found")
	ErrCampaignRequired     = errors.New("chat: campaign_id is required")
	ErrMessageTooShort      = errors.New("chat: message too short")
	ErrMessageTooLong       = errors.New("chat: message too long")
	ErrLLMFailed            = errors.New("chat: LLM generation failed")
	ErrSearchFailed         = errors.New("chat: search failed")
	ErrConversationArchived = errors.New("chat: conversation is archived")
)
