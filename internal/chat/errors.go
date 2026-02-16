package chat

import "errors"

// Domain errors
var (
	// ErrConversationNotFound - Conversation không tồn tại
	ErrConversationNotFound = errors.New("chat: conversation not found")

	// ErrCampaignRequired - Campaign ID bắt buộc
	ErrCampaignRequired = errors.New("chat: campaign_id is required")

	// ErrMessageTooShort - Message quá ngắn (< 3 chars)
	ErrMessageTooShort = errors.New("chat: message too short")

	// ErrMessageTooLong - Message quá dài (> 2000 chars)
	ErrMessageTooLong = errors.New("chat: message too long")

	// ErrLLMFailed - LLM generation thất bại
	ErrLLMFailed = errors.New("chat: LLM generation failed")

	// ErrSearchFailed - Search thất bại
	ErrSearchFailed = errors.New("chat: search failed")

	// ErrConversationArchived - Conversation đã archived
	ErrConversationArchived = errors.New("chat: conversation is archived")
)
