package http

import (
	"knowledge-srv/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes - Register HTTP routes
func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware) {
	// Internal API group (service-to-service auth)
	internal := r.Group("/internal")
	internal.Use(mw.ServiceAuth()) // Service token authentication
	{
		internal.POST("/index/by-file", h.IndexByFile)
	}
}
