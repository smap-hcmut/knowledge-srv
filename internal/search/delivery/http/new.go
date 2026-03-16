package http

import (
	"knowledge-srv/internal/search"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/middleware"
)

// Handler - Interface cho search HTTP handler
type Handler interface {
	RegisterRoutes(r *gin.RouterGroup, mw *middleware.Middleware)
}

type handler struct {
	l       log.Logger
	uc      search.UseCase
	discord discord.IDiscord
}

// New - Factory
func New(l log.Logger, uc search.UseCase, discord discord.IDiscord) Handler {
	return &handler{l: l, uc: uc, discord: discord}
}
