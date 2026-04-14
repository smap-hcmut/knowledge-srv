package httpserver

import (
	"context"
	chatHTTP "knowledge-srv/internal/chat/delivery/http"
	chatPostgre "knowledge-srv/internal/chat/repository/postgre"
	chatUsecase "knowledge-srv/internal/chat/usecase"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/middleware"
)

func (srv *HTTPServer) setupChatDomain(ctx context.Context, r *gin.RouterGroup, mw *middleware.Middleware) error {
	repo := chatPostgre.New(srv.postgresDB, srv.l)

	uc := chatUsecase.New(repo, srv.searchUC, srv.llmClient, srv.l)

	handler := chatHTTP.New(srv.l, uc, srv.discord)
	handler.RegisterRoutes(r, mw)

	srv.l.Infof(ctx, "Chat domain registered")
	return nil
}
