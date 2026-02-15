package usecase

import (
	"knowledge-srv/internal/indexing"
	"knowledge-srv/pkg/discord"
	"knowledge-srv/pkg/gemini"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/minio"
	"knowledge-srv/pkg/qdrant"
	"knowledge-srv/pkg/redis"
	"knowledge-srv/pkg/voyage"
)

// implUseCase implements the indexing.UseCase interface
type implUseCase struct {
	l        log.Logger
	repo     indexing.Repository
	qdrant   qdrant.IQdrant
	minio    minio.MinIO
	voyage   voyage.IVoyage
	gemini   gemini.IGemini
	redis    redis.IRedis
	producer indexing.Producer
	discord  discord.IDiscord
}

// New creates a new indexing usecase
func New(
	l log.Logger,
	repo indexing.Repository,
	qdrant qdrant.IQdrant,
	minio minio.MinIO,
	voyage voyage.IVoyage,
	gemini gemini.IGemini,
	redis redis.IRedis,
	producer indexing.Producer,
	discord discord.IDiscord,
) indexing.UseCase {
	return &implUseCase{
		l:        l,
		repo:     repo,
		qdrant:   qdrant,
		minio:    minio,
		voyage:   voyage,
		gemini:   gemini,
		redis:    redis,
		producer: producer,
		discord:  discord,
	}
}
