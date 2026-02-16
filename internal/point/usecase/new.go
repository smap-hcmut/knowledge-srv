package usecase

import (
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
	"knowledge-srv/pkg/log"
)

type implUseCase struct {
	repo repository.QdrantRepository
	l    log.Logger
}

func New(repo repository.QdrantRepository, l log.Logger) point.UseCase {
	return &implUseCase{
		repo: repo,
		l:    l,
	}
}
