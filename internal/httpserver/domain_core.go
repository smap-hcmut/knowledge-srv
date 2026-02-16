package httpserver

import (
	"context"

	embeddingRepo "knowledge-srv/internal/embedding/repository/redis"
	embeddingUsecase "knowledge-srv/internal/embedding/usecase"
	pointRepo "knowledge-srv/internal/point/repository/qdrant"
	pointUsecase "knowledge-srv/internal/point/usecase"
)

// setupCoreDomains initializes shared domains (embedding, point)
func (srv *HTTPServer) setupCoreDomains(ctx context.Context) error {
	// 1. Embedding Domain
	// Repositories
	embeddingCacheRepo := embeddingRepo.New(srv.redisClient, srv.l)

	// UseCase
	srv.embeddingUC = embeddingUsecase.New(
		embeddingCacheRepo, // 1st arg: CacheRepository
		srv.voyageClient,   // 2nd arg: Voyage
		srv.l,
	)

	// 2. Point Domain
	// Repositories
	pointQdrantRepo := pointRepo.New(srv.qdrantClient, srv.l)

	// UseCase
	srv.pointUC = pointUsecase.New(
		pointQdrantRepo,
		srv.l,
	)

	srv.l.Infof(ctx, "Core domains (Embedding, Point) initialized")
	return nil
}
