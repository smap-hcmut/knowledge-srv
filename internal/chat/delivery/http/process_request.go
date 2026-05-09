package http

import (
	"knowledge-srv/internal/model"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/auth"
	pkgErrors "github.com/smap-hcmut/shared-libs/go/errors"
)

func (h *handler) processChatRequest(c *gin.Context) (chatReq, model.Scope, error) {
	var req chatReq

	if err := c.ShouldBindJSON(&req); err != nil {
		if strings.TrimSpace(req.CampaignID) == "" {
			return req, model.Scope{}, errCampaignRequired
		}
		if strings.TrimSpace(req.Message) == "" {
			return req, model.Scope{}, errMessageTooShort
		}
		return req, model.Scope{}, pkgErrors.NewHTTPError(400, "Invalid chat request")
	}
	req.CampaignID = strings.TrimSpace(req.CampaignID)
	req.Message = strings.TrimSpace(req.Message)
	req.ConversationID = strings.TrimSpace(req.ConversationID)

	sc := auth.GetScopeFromContext(c.Request.Context())
	return req, model.ToScope(sc), nil
}

func (h *handler) processGetConversationRequest(c *gin.Context) (getConversationReq, model.Scope, error) {
	req := getConversationReq{
		ConversationID: c.Param("conversation_id"),
	}

	sc := auth.GetScopeFromContext(c.Request.Context())
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

	sc := auth.GetScopeFromContext(c.Request.Context())
	return req, model.ToScope(sc), nil
}

func (h *handler) processGetSuggestionsRequest(c *gin.Context) (getSuggestionsReq, model.Scope, error) {
	req := getSuggestionsReq{
		CampaignID: c.Param("campaign_id"),
	}

	sc := auth.GetScopeFromContext(c.Request.Context())
	return req, model.ToScope(sc), nil
}
