package usecase

import (
	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/search"

	"github.com/smap-hcmut/shared-libs/go/llm"
	"github.com/smap-hcmut/shared-libs/go/log"
)

type implUseCase struct {
	repo     repository.PostgresRepository
	searchUC search.UseCase
	llm      llm.LLM
	l        log.Logger
}

func New(
	repo repository.PostgresRepository,
	searchUC search.UseCase,
	llmClient llm.LLM,
	l log.Logger,
) chat.UseCase {
	return &implUseCase{
		repo:     repo,
		searchUC: searchUC,
		llm:      llmClient,
		l:        l,
	}
}
