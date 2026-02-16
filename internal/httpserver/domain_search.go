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

// setupSearchDomain initializes search domain (repo -> usecase -> delivery)
func (srv *HTTPServer) setupSearchDomain(ctx context.Context, r *gin.RouterGroup, mw middleware.Middleware) error {
	// Repositories (Only cache repo needed for search specific caches)
	cacheRepo := searchRedis.New(srv.redisClient, srv.l)

	// Project Service client
	projectSrv := projectsrv.New(projectsrv.ProjectConfig{
		BaseURL: srv.config.Project.URL,
	})

	// UseCase
	cfg := searchUsecase.DefaultConfig()
	uc := searchUsecase.New(
		srv.pointUC,
		srv.embeddingUC,
		cacheRepo,
		projectSrv,
		srv.l,
		cfg,
	)

	// HTTP Handler
	handler := searchHTTP.New(srv.l, uc, srv.discord)

	// Register routes
	handler.RegisterRoutes(r, mw)

	srv.l.Infof(ctx, "Search domain registered")
	return nil
}
