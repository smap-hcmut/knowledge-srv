package http

import (
	"github.com/gin-gonic/gin"

	"knowledge-srv/internal/indexing"
	"knowledge-srv/pkg/discord"
	"knowledge-srv/pkg/log"
)

type Handler interface {
	Index(c *gin.Context)
	RetryFailed(c *gin.Context)
	Reconcile(c *gin.Context)
	GetStatistics(c *gin.Context)
}

type handler struct {
	l       log.Logger
	uc      indexing.UseCase
	discord discord.IDiscord
}

func New(l log.Logger, uc indexing.UseCase, d discord.IDiscord) Handler {
	return &handler{
		l:       l,
		uc:      uc,
		discord: d,
	}
}
