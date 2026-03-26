package http

import (
	"errors"

	"knowledge-srv/internal/chat"

	pkgErrors "github.com/smap-hcmut/shared-libs/go/errors"
)

var (
	errConversationNotFound = pkgErrors.NewHTTPError(404, "Conversation not found")
	errJobIDRequired        = pkgErrors.NewHTTPError(400, "job_id is required")
	errCampaignRequired     = pkgErrors.NewHTTPError(400, "Campaign ID is required")
	errMessageTooShort      = pkgErrors.NewHTTPError(400, "Message too short (min 3 characters)")
	errMessageTooLong       = pkgErrors.NewHTTPError(400, "Message too long (max 2000 characters)")
	errLLMFailed            = pkgErrors.NewHTTPError(500, "AI generation failed")
	errSearchFailed         = pkgErrors.NewHTTPError(500, "Search failed")
	errConversationArchived = pkgErrors.NewHTTPError(400, "Conversation is archived")
)

func (h *handler) mapError(err error) error {
	switch {
	case errors.Is(err, chat.ErrConversationNotFound):
		return errConversationNotFound
	case errors.Is(err, chat.ErrCampaignRequired):
		return errCampaignRequired
	case errors.Is(err, chat.ErrMessageTooShort):
		return errMessageTooShort
	case errors.Is(err, chat.ErrMessageTooLong):
		return errMessageTooLong
	case errors.Is(err, chat.ErrLLMFailed):
		return errLLMFailed
	case errors.Is(err, chat.ErrSearchFailed):
		return errSearchFailed
	case errors.Is(err, chat.ErrConversationArchived):
		return errConversationArchived
	case errors.Is(err, chat.ErrChatJobNotFound):
		return pkgErrors.NewHTTPError(404, "Chat job not found")
	default:
		return err
	}
}
