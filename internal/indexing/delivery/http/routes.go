package http

import (
	"knowledge-srv/internal/middleware"

	"github.com/gin-gonic/gin"
)

func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware) {
	internal := r.Group("/internal")
	internal.Use(mw.ServiceAuth())
	{
		internal.POST("/index", h.Index)
	}
}
