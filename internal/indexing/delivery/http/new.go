package http

import (
	"knowledge-srv/internal/indexing"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/log"
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
