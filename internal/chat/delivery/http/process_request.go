package http

import (
	"knowledge-srv/internal/model"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/scope"
)

func (h *handler) processChatRequest(c *gin.Context) (chatReq, model.Scope, error) {
	var req chatReq

	if err := c.ShouldBindJSON(&req); err != nil {
		return req, model.Scope{}, err
	}

	sc, _ := scope.GetScopeFromContext(c.Request.Context())
	return req, model.ToScope(sc), nil
}

func (h *handler) processGetConversationRequest(c *gin.Context) (getConversationReq, model.Scope, error) {
	req := getConversationReq{
		ConversationID: c.Param("conversation_id"),
	}

	sc, _ := scope.GetScopeFromContext(c.Request.Context())
	return req, model.ToScope(sc), nil
}

func (h *handler) processListConversationsRequest(c *gin.Context) (listConversationsReq, model.Scope, error) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	req := listConversationsReq{
		CampaignID: c.Param("campaign_id"),
		Limit:      limit,
		Offset:     offset,
	}

	sc, _ := scope.GetScopeFromContext(c.Request.Context())
	return req, model.ToScope(sc), nil
}

func (h *handler) processGetSuggestionsRequest(c *gin.Context) (getSuggestionsReq, model.Scope, error) {
	req := getSuggestionsReq{
		CampaignID: c.Param("campaign_id"),
	}

	sc, _ := scope.GetScopeFromContext(c.Request.Context())
	return req, model.ToScope(sc), nil
}
