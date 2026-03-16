package http

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/middleware"
)

func (h *handler) RegisterRoutes(r *gin.RouterGroup, mw *middleware.Middleware) {
	r.Use(mw.Auth())
	{
		r.POST("/search", h.Search)
	}
}
