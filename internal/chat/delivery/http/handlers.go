package http

import (
	"knowledge-srv/pkg/response"

	"github.com/gin-gonic/gin"
)

// Chat - Handler cho POST /api/v1/chat
func (h *handler) Chat(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processChatRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "processChatRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	input := req.toInput()

	output, err := h.uc.Chat(ctx, sc, input)
	if err != nil {
		h.l.Errorf(ctx, "Chat failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	resp := h.newChatResp(output)
	response.OK(c, resp)
}

// GetConversation - Handler cho GET /api/v1/conversations/:conversation_id
func (h *handler) GetConversation(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processGetConversationRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "processGetConversationRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	output, err := h.uc.GetConversation(ctx, sc, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "GetConversation failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	resp := h.newConversationResp(output)
	response.OK(c, resp)
}

// ListConversations - Handler cho GET /api/v1/campaigns/:campaign_id/conversations
func (h *handler) ListConversations(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processListConversationsRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "processListConversationsRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	output, err := h.uc.ListConversations(ctx, sc, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "ListConversations failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	resp := h.newListConversationsResp(output)
	response.OK(c, resp)
}

// GetSuggestions - Handler cho GET /api/v1/campaigns/:campaign_id/suggestions
func (h *handler) GetSuggestions(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processGetSuggestionsRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "processGetSuggestionsRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	output, err := h.uc.GetSuggestions(ctx, sc, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "GetSuggestions failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	resp := h.newSuggestionsResp(output)
	response.OK(c, resp)
}
