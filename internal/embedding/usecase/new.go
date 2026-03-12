package usecase

import (
	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/embedding/repository"
	"knowledge-srv/pkg/voyage"

	"github.com/smap-hcmut/shared-libs/go/log"
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
