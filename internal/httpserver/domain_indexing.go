package httpserver

import (
	"context"

	"github.com/gin-gonic/gin"

	indexingHTTP "knowledge-srv/internal/indexing/delivery/http"
	indexingPostgre "knowledge-srv/internal/indexing/repository/postgre"
	indexingRedis "knowledge-srv/internal/indexing/repository/redis"
	indexingUsecase "knowledge-srv/internal/indexing/usecase"
	"knowledge-srv/internal/middleware"
)

// setupIndexingDomain initializes indexing domain (repo -> usecase -> delivery)
func (srv *HTTPServer) setupIndexingDomain(ctx context.Context, r *gin.RouterGroup, mw middleware.Middleware) error {
	// Repositories
	postgreRepo := indexingPostgre.New(srv.postgresDB, srv.l)
	cacheRepo := indexingRedis.New(srv.redisClient, srv.l)

	// UseCase
	uc := indexingUsecase.New(
		srv.l,
		postgreRepo,
		srv.pointUC,
		srv.embeddingUC,
		cacheRepo,
		srv.minioClient,
	)

	// HTTP Handler
	handler := indexingHTTP.New(srv.l, uc, srv.discord)

	// Register routes
	handler.(interface {
		RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware)
	}).RegisterRoutes(r, mw)

	srv.l.Infof(ctx, "Indexing domain registered")
	return nil
}
