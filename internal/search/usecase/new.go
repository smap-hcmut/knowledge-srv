package usecase

import (
	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/search"
	"knowledge-srv/internal/search/repository"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/projectsrv"
)

// Config - Cấu hình UseCase
type Config struct {
	MinScore       float64 // Min relevance score threshold (default 0.65)
	MaxResults     int     // Max results per query (default 10, max 50)
	MinQueryLength int     // Min query length in chars (default 3)
	MaxQueryLength int     // Max query length in chars (default 1000)
}

// DefaultConfig - Cấu hình mặc định
func DefaultConfig() Config {
	return Config{
		MinScore:       0.65,
		MaxResults:     10,
		MinQueryLength: 3,
		MaxQueryLength: 1000,
	}
}

// implUseCase - Implementation của UseCase interface
type implUseCase struct {
	pointUC     point.UseCase
	embeddingUC embedding.UseCase
	cacheRepo   repository.CacheRepository
	projectSrv  projectsrv.IProject
	l           log.Logger
	cfg         Config
}

// New - Factory function
func New(
	pointUC point.UseCase,
	embeddingUC embedding.UseCase,
	cacheRepo repository.CacheRepository,
	projectSrv projectsrv.IProject,
	l log.Logger,
	cfg Config,
) search.UseCase {
	return &implUseCase{
		pointUC:     pointUC,
		embeddingUC: embeddingUC,
		cacheRepo:   cacheRepo,
		projectSrv:  projectSrv,
		l:           l,
		cfg:         cfg,
	}
}
