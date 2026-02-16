package httpserver

import (
	"context"

	embeddingRepo "knowledge-srv/internal/embedding/repository/redis"
	embeddingUsecase "knowledge-srv/internal/embedding/usecase"
	pointRepo "knowledge-srv/internal/point/repository/qdrant"
	pointUsecase "knowledge-srv/internal/point/usecase"
)

func (srv *HTTPServer) setupCoreDomains(ctx context.Context) error {
	embeddingCacheRepo := embeddingRepo.New(srv.redisClient, srv.l)

	srv.embeddingUC = embeddingUsecase.New(embeddingCacheRepo, srv.voyageClient, srv.l)

	pointQdrantRepo := pointRepo.New(srv.qdrantClient, srv.l)

	srv.pointUC = pointUsecase.New(pointQdrantRepo, srv.l)

	srv.l.Infof(ctx, "Core domains (Embedding, Point) initialized")
	return nil
}
