package http

import (
	"strconv"

	"knowledge-srv/internal/model"
	"knowledge-srv/pkg/scope"

	"github.com/gin-gonic/gin"
)

// processChatRequest - Bind + validate POST /api/v1/chat
func (h *handler) processChatRequest(c *gin.Context) (chatReq, model.Scope, error) {
	var req chatReq

	if err := c.ShouldBindJSON(&req); err != nil {
		return req, model.Scope{}, err
	}

	sc := scope.GetScopeFromContext(c.Request.Context())
	return req, sc, nil
}

// processGetConversationRequest - Extract conversation_id from path
func (h *handler) processGetConversationRequest(c *gin.Context) (getConversationReq, model.Scope, error) {
	req := getConversationReq{
		ConversationID: c.Param("conversation_id"),
	}

	sc := scope.GetScopeFromContext(c.Request.Context())
	return req, sc, nil
}

// processListConversationsRequest - Extract campaign_id from path + pagination from query
func (h *handler) processListConversationsRequest(c *gin.Context) (listConversationsReq, model.Scope, error) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	req := listConversationsReq{
		CampaignID: c.Param("campaign_id"),
		Limit:      limit,
		Offset:     offset,
	}

	sc := scope.GetScopeFromContext(c.Request.Context())
	return req, sc, nil
}

// processGetSuggestionsRequest - Extract campaign_id from path
func (h *handler) processGetSuggestionsRequest(c *gin.Context) (getSuggestionsReq, model.Scope, error) {
	req := getSuggestionsReq{
		CampaignID: c.Param("campaign_id"),
	}

	sc := scope.GetScopeFromContext(c.Request.Context())
	return req, sc, nil
}
