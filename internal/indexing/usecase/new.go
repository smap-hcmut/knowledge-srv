package usecase

import (
	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/indexing"
	repo "knowledge-srv/internal/indexing/repository"
	"knowledge-srv/internal/point"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/minio"
)

// implUseCase implements the indexing.UseCase interface
type implUseCase struct {
	l           log.Logger
	postgreRepo repo.PostgresRepository
	pointUC     point.UseCase
	embeddingUC embedding.UseCase
	cacheRepo   repo.CacheRepository
	minio       minio.MinIO
}

// New creates a new indexing usecase.
func New(
	l log.Logger,
	postgreRepo repo.PostgresRepository,
	pointUC point.UseCase,
	embeddingUC embedding.UseCase,
	cacheRepo repo.CacheRepository,
	minio minio.MinIO,
) indexing.UseCase {
	return &implUseCase{
		l:           l,
		postgreRepo: postgreRepo,
		pointUC:     pointUC,
		embeddingUC: embeddingUC,
		cacheRepo:   cacheRepo,
		minio:       minio,
	}
}
