package http

import (
	"knowledge-srv/internal/search"
	"knowledge-srv/pkg/discord"
	"knowledge-srv/pkg/log"

	"knowledge-srv/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Handler - Interface cho search HTTP handler
type Handler interface {
	RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware)
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
