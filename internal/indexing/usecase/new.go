package usecase

import (
	"knowledge-srv/internal/indexing"
	repo "knowledge-srv/internal/indexing/repository"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/minio"
	"knowledge-srv/pkg/redis"
	"knowledge-srv/pkg/voyage"
)

// implUseCase implements the indexing.UseCase interface
type implUseCase struct {
	l           log.Logger
	postgreRepo repo.PostgresRepository
	vectorRepo  repo.QdrantRepository
	minio       minio.MinIO
	voyage      voyage.IVoyage
	redis       redis.IRedis
}

// New creates a new indexing usecase.
func New(
	l log.Logger,
	postgreRepo repo.PostgresRepository,
	vectorRepo repo.QdrantRepository,
	minio minio.MinIO,
	voyage voyage.IVoyage,
	redis redis.IRedis,
) indexing.UseCase {
	return &implUseCase{
		l:           l,
		postgreRepo: postgreRepo,
		vectorRepo:  vectorRepo,
		minio:       minio,
		voyage:      voyage,
		redis:       redis,
	}
}
