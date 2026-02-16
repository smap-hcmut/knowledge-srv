package httpserver

import (
	"context"

	"github.com/gin-gonic/gin"

	indexingHTTP "knowledge-srv/internal/indexing/delivery/http"
	indexingPostgre "knowledge-srv/internal/indexing/repository/postgre"
	indexingQdrant "knowledge-srv/internal/indexing/repository/qdrant"
	indexingRedis "knowledge-srv/internal/indexing/repository/redis"
	indexingUsecase "knowledge-srv/internal/indexing/usecase"
	"knowledge-srv/internal/middleware"
)

// setupIndexingDomain initializes indexing domain (repo -> usecase -> delivery)
func (srv *HTTPServer) setupIndexingDomain(ctx context.Context, r *gin.RouterGroup, mw middleware.Middleware) error {
	// Repositories (collection name là const trong qdrant package)
	postgreRepo := indexingPostgre.New(srv.postgresDB, srv.l)
	vectorRepo := indexingQdrant.New(srv.qdrantClient, srv.l)
	cacheRepo := indexingRedis.New(srv.redisClient, srv.l)

	// UseCase
	uc := indexingUsecase.New(
		srv.l,
		postgreRepo,
		vectorRepo,
		cacheRepo,
		srv.minioClient,
		srv.voyageClient,
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
