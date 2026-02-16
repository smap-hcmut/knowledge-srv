package http

import (
	"knowledge-srv/internal/middleware"

	"github.com/gin-gonic/gin"
)

func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware) {
	api := r.Group("/api/v1")
	api.Use(mw.Auth())
	{
		// Chat
		api.POST("/chat", h.Chat)

		// Conversations
		api.GET("/conversations/:conversation_id", h.GetConversation)

		// Campaign-scoped
		api.GET("/campaigns/:campaign_id/conversations", h.ListConversations)
		api.GET("/campaigns/:campaign_id/suggestions", h.GetSuggestions)
	}
}
