package usecase

import (
	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/search"
	"knowledge-srv/pkg/gemini"
	"knowledge-srv/pkg/log"
)

type implUseCase struct {
	repo     repository.PostgresRepository
	searchUC search.UseCase
	gemini   gemini.IGemini
	l        log.Logger
}

// New - Factory function
func New(
	repo repository.PostgresRepository,
	searchUC search.UseCase,
	gemini gemini.IGemini,
	l log.Logger,
) chat.UseCase {
	return &implUseCase{
		repo:     repo,
		searchUC: searchUC,
		gemini:   gemini,
		l:        l,
	}
}
