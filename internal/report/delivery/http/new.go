package http

import (
	"knowledge-srv/internal/middleware"
	"knowledge-srv/internal/report"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/log"
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
