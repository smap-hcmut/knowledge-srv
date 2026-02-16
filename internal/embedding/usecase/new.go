package usecase

import (
	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/embedding/repository"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/voyage"
)

type implUseCase struct {
	repo   repository.Repository
	voyage voyage.IVoyage
	l      log.Logger
}

func New(repo repository.Repository, voyage voyage.IVoyage, l log.Logger) embedding.UseCase {
	return &implUseCase{
		repo:   repo,
		voyage: voyage,
		l:      l,
	}
}
