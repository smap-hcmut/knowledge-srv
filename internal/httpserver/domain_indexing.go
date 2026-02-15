package httpserver

import (
	"context"

	"github.com/gin-gonic/gin"

	indexingHTTP "knowledge-srv/internal/indexing/delivery/http"
	indexingRepo "knowledge-srv/internal/indexing/repository/postgre"
	indexingUsecase "knowledge-srv/internal/indexing/usecase"
	"knowledge-srv/internal/middleware"
)

// setupIndexingDomain initializes indexing domain (repo -> usecase -> delivery)
func (srv *HTTPServer) setupIndexingDomain(ctx context.Context, r *gin.RouterGroup, mw middleware.Middleware) error {
	// Repository
	repo := indexingRepo.New(srv.postgresDB)

	// UseCase
	uc := indexingUsecase.New(
		srv.l,
		repo,
		srv.qdrantClient,
		srv.minioClient,
		srv.voyageClient,
		srv.redisClient,
		"knowledge_indexing", // TODO: move to config
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
