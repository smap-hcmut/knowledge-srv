package http

import (
	"github.com/gin-gonic/gin"

	"knowledge-srv/internal/indexing"
	"knowledge-srv/pkg/discord"
	"knowledge-srv/pkg/log"
)

// Handler defines the HTTP handler interface
type Handler interface {
	IndexByFile(c *gin.Context)
}

// handler - HTTP handler implementation
type handler struct {
	l       log.Logger
	uc      indexing.UseCase
	discord discord.IDiscord
}

// New creates a new HTTP handler
func New(l log.Logger, uc indexing.UseCase, d discord.IDiscord) Handler {
	return &handler{
		l:       l,
		uc:      uc,
		discord: d,
	}
}
