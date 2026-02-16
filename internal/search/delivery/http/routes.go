package http

import (
	"knowledge-srv/internal/middleware"

	"github.com/gin-gonic/gin"
)

func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware) {
	api := r.Group("/api/v1")
	api.Use(mw.Auth())
	{
		api.POST("/search", h.Search)
	}
}
