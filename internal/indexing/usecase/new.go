package usecase

import (
	"knowledge-srv/internal/indexing"
	repo "knowledge-srv/internal/indexing/repository"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/minio"
	"knowledge-srv/pkg/qdrant"
	"knowledge-srv/pkg/redis"
	"knowledge-srv/pkg/voyage"
)

// implUseCase implements the indexing.UseCase interface
type implUseCase struct {
	l      log.Logger
	repo   repo.Repository
	qdrant qdrant.IQdrant
	minio  minio.MinIO
	voyage voyage.IVoyage
	redis  redis.IRedis

	// Config
	collectionName   string
	maxConcurrency   int
	minContentLength int
	minQualityScore  float64
}

// New creates a new indexing usecase
func New(
	l log.Logger,
	repository repo.Repository,
	qdrant qdrant.IQdrant,
	minio minio.MinIO,
	voyage voyage.IVoyage,
	redis redis.IRedis,
	collectionName string,
) indexing.UseCase {
	return &implUseCase{
		l:                l,
		repo:             repository,
		qdrant:           qdrant,
		minio:            minio,
		voyage:           voyage,
		redis:            redis,
		collectionName:   collectionName,
		maxConcurrency:   10,  // Parallel workers
		minContentLength: 10,  // Min chars
		minQualityScore:  0.3, // Min quality score
	}
}
