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

func (srv *HTTPServer) setupIndexingDomain(ctx context.Context, r *gin.RouterGroup, mw middleware.Middleware) error {
	postgreRepo := indexingPostgre.New(srv.postgresDB, srv.l)
	cacheRepo := indexingRedis.New(srv.redisClient, srv.l)

	uc := indexingUsecase.New(srv.l, postgreRepo, srv.pointUC, srv.embeddingUC, cacheRepo, srv.minioClient)

	handler := indexingHTTP.New(srv.l, uc, srv.discord)
	handler.(interface {
		RegisterRoutes(r *gin.RouterGroup, mw middleware.Middleware)
	}).RegisterRoutes(r, mw)

	srv.l.Infof(ctx, "Indexing domain registered")
	return nil
}
