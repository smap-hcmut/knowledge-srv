package httpserver

import (
	"context"

	"github.com/gin-gonic/gin"

	"knowledge-srv/internal/middleware"
	searchHTTP "knowledge-srv/internal/search/delivery/http"
	searchRedis "knowledge-srv/internal/search/repository/redis"
	searchUsecase "knowledge-srv/internal/search/usecase"
	"knowledge-srv/pkg/projectsrv"
)

func (srv *HTTPServer) setupSearchDomain(ctx context.Context, r *gin.RouterGroup, mw middleware.Middleware) error {
	cacheRepo := searchRedis.New(srv.redisClient, srv.l)

	projectSrv := projectsrv.New(projectsrv.ProjectConfig{
		BaseURL: srv.config.Project.URL,
	})

	uc := searchUsecase.New(srv.pointUC, srv.embeddingUC, cacheRepo, projectSrv, srv.l)
	srv.searchUC = uc

	handler := searchHTTP.New(srv.l, uc, srv.discord)
	handler.RegisterRoutes(r, mw)

	srv.l.Infof(ctx, "Search domain registered")
	return nil
}
