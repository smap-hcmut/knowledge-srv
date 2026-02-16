package http

import (
	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/middleware"
	"knowledge-srv/pkg/discord"
	"knowledge-srv/pkg/log"

	"github.com/gin-gonic/gin"
)

// Handler - Interface cho chat HTTP handler
type Handler interface {
	RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware)
}

type handler struct {
	l       log.Logger
	uc      chat.UseCase
	discord discord.IDiscord
}

// New - Factory
func New(l log.Logger, uc chat.UseCase, discord discord.IDiscord) Handler {
	return &handler{l: l, uc: uc, discord: discord}
}
