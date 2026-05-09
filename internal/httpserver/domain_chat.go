package httpserver

import (
	"context"
	chatHTTP "knowledge-srv/internal/chat/delivery/http"
	chatPostgre "knowledge-srv/internal/chat/repository/postgre"
	chatUsecase "knowledge-srv/internal/chat/usecase"
	"knowledge-srv/pkg/analytics"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/middleware"
)

func (srv *HTTPServer) setupChatDomain(ctx context.Context, r *gin.RouterGroup, mw *middleware.Middleware) error {
	repo := chatPostgre.New(srv.postgresDB, srv.l)
	analyticsClient := analytics.New(analytics.Config{
		BaseURL: srv.config.Analysis.URL,
		Timeout: time.Duration(srv.config.Analysis.Timeout) * time.Second,
	})

	uc := chatUsecase.New(repo, srv.searchUC, analyticsClient, srv.llmClient, srv.l)

	handler := chatHTTP.New(srv.l, uc, srv.discord)
	handler.RegisterRoutes(r, mw)

	srv.l.Infof(ctx, "Chat domain registered")
	return nil
}
