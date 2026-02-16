package http

import (
	"knowledge-srv/internal/middleware"
	"knowledge-srv/internal/report"
	"knowledge-srv/pkg/discord"
	"knowledge-srv/pkg/log"

	"github.com/gin-gonic/gin"
)

type Handler interface {
	RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware)
}

type handler struct {
	l       log.Logger
	uc      report.UseCase
	discord discord.IDiscord
}

func New(l log.Logger, uc report.UseCase, discord discord.IDiscord) Handler {
	return &handler{
		l:       l,
		uc:      uc,
		discord: discord,
	}
}
