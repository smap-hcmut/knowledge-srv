package usecase

import (
	"knowledge-srv/internal/indexing"
	repo "knowledge-srv/internal/indexing/repository"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/minio"
	"knowledge-srv/pkg/voyage"
)

// implUseCase implements the indexing.UseCase interface
type implUseCase struct {
	l           log.Logger
	postgreRepo repo.PostgresRepository
	vectorRepo  repo.QdrantRepository
	cacheRepo   repo.CacheRepository
	minio       minio.MinIO
	voyage      voyage.IVoyage
}

// New creates a new indexing usecase.
func New(
	l log.Logger,
	postgreRepo repo.PostgresRepository,
	vectorRepo repo.QdrantRepository,
	cacheRepo repo.CacheRepository,
	minio minio.MinIO,
	voyage voyage.IVoyage,
) indexing.UseCase {
	return &implUseCase{
		l:           l,
		postgreRepo: postgreRepo,
		vectorRepo:  vectorRepo,
		cacheRepo:   cacheRepo,
		minio:       minio,
		voyage:      voyage,
	}
}
