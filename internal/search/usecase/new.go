package usecase

import (
	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/search"
	"knowledge-srv/internal/search/repository"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/projectsrv"
)

// implUseCase - Implementation của UseCase interface
type implUseCase struct {
	pointUC     point.UseCase
	embeddingUC embedding.UseCase
	cacheRepo   repository.CacheRepository
	projectSrv  projectsrv.IProject
	l           log.Logger
}

// New - Factory function
func New(
	pointUC point.UseCase,
	embeddingUC embedding.UseCase,
	cacheRepo repository.CacheRepository,
	projectSrv projectsrv.IProject,
	l log.Logger,
) search.UseCase {
	return &implUseCase{
		pointUC:     pointUC,
		embeddingUC: embeddingUC,
		cacheRepo:   cacheRepo,
		projectSrv:  projectSrv,
		l:           l,
	}
}
