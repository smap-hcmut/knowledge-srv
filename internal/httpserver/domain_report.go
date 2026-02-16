package httpserver

import (
	"context"

	"github.com/gin-gonic/gin"

	"knowledge-srv/internal/middleware"
	reportHTTP "knowledge-srv/internal/report/delivery/http"
	reportPostgre "knowledge-srv/internal/report/repository/postgre"
	reportUsecase "knowledge-srv/internal/report/usecase"
)

func (srv *HTTPServer) setupReportDomain(ctx context.Context, r *gin.RouterGroup, mw middleware.Middleware) error {
	repo := reportPostgre.New(srv.postgresDB, srv.l)

	uc := reportUsecase.New(repo, srv.searchUC, srv.geminiClient, srv.minioClient, srv.l, reportUsecase.Config{})

	handler := reportHTTP.New(srv.l, uc, srv.discord)
	handler.RegisterRoutes(r, mw)

	srv.l.Infof(ctx, "Report domain registered")
	return nil
}
