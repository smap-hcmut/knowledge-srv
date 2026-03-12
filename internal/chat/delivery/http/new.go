package http

import (
	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/log"
)

type Handler interface {
	RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware)
}

type handler struct {
	l       log.Logger
	uc      chat.UseCase
	discord discord.IDiscord
}

func New(l log.Logger, uc chat.UseCase, discord discord.IDiscord) Handler {
	return &handler{
		l:       l,
		uc:      uc,
		discord: discord,
	}
}
