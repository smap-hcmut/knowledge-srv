package http

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/middleware"
)

func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw *middleware.Middleware) {
	r.Use(mw.Auth())
	{
		r.POST("/chat", h.Chat)
		r.GET("/conversations/:conversation_id", h.GetConversation)
		r.GET("/campaigns/:campaign_id/conversations", h.ListConversations)
		r.GET("/campaigns/:campaign_id/suggestions", h.GetSuggestions)
	}
}
