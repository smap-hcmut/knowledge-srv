package http

import (
	"knowledge-srv/pkg/response"

	"github.com/gin-gonic/gin"
)

// @Summary Chat with knowledge service
// @Description Send a message and receive an answer with citations
// @Tags Chat
// @Accept json
// @Produce json
// @Param body body chatReq true "Chat request"
// @Success 200 {object} chatResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /api/v1/chat [post]
func (h *handler) Chat(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processChatRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "chat.delivery.http.Chat: processChatRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.Chat(ctx, sc, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "chat.delivery.http.Chat: usecase Chat failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newChatResp(o))
}

// @Summary Get conversation detail
// @Description Return full conversation info and messages
// @Tags Chat
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} conversationResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /api/v1/conversations/{conversation_id} [get]
func (h *handler) GetConversation(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processGetConversationRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "chat.delivery.http.GetConversation: processGetConversationRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.GetConversation(ctx, sc, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "chat.delivery.http.GetConversation: usecase GetConversation failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newConversationResp(o))
}

// @Summary List conversations by campaign
// @Description Paginate conversations for a given campaign
// @Tags Chat
// @Accept json
// @Produce json
// @Param campaign_id path string true "Campaign ID"
// @Param limit query int false "Number of records per page (default 20)"
// @Param offset query int false "Number of records to skip (default 0)"
// @Success 200 {array} conversationResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /api/v1/campaigns/{campaign_id}/conversations [get]
func (h *handler) ListConversations(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processListConversationsRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "chat.delivery.http.ListConversations: processListConversationsRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.ListConversations(ctx, sc, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "chat.delivery.http.ListConversations: usecase ListConversations failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newListConversationsResp(o))
}

// @Summary Get smart suggestions
// @Description Return a list of suggested queries for a campaign
// @Tags Chat
// @Accept json
// @Produce json
// @Param campaign_id path string true "Campaign ID"
// @Success 200 {object} suggestionsResp
// @Failure 400 {object} response.Resp
// @Failure 500 {object} response.Resp
// @Router /api/v1/campaigns/{campaign_id}/suggestions [get]
func (h *handler) GetSuggestions(c *gin.Context) {
	ctx := c.Request.Context()

	req, sc, err := h.processGetSuggestionsRequest(c)
	if err != nil {
		h.l.Errorf(ctx, "chat.delivery.http.GetSuggestions: processGetSuggestionsRequest failed: %v", err)
		response.Error(c, err, h.discord)
		return
	}

	o, err := h.uc.GetSuggestions(ctx, sc, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "chat.delivery.http.GetSuggestions: usecase GetSuggestions failed: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newSuggestionsResp(o))
}
