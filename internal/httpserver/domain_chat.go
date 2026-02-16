package httpserver

import (
	"context"

	"github.com/gin-gonic/gin"

	chatHTTP "knowledge-srv/internal/chat/delivery/http"
	chatPostgre "knowledge-srv/internal/chat/repository/postgre"
	chatUsecase "knowledge-srv/internal/chat/usecase"
	"knowledge-srv/internal/middleware"
)

func (srv *HTTPServer) setupChatDomain(ctx context.Context, r *gin.RouterGroup, mw middleware.Middleware) error {
	repo := chatPostgre.New(srv.postgresDB, srv.l)

	uc := chatUsecase.New(repo, srv.searchUC, srv.geminiClient, srv.l)

	handler := chatHTTP.New(srv.l, uc, srv.discord)
	handler.RegisterRoutes(r, mw)

	srv.l.Infof(ctx, "Chat domain registered")
	return nil
}
