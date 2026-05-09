package usecase

import (
	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/search"
	"knowledge-srv/pkg/analytics"

	"github.com/smap-hcmut/shared-libs/go/llm"
	"github.com/smap-hcmut/shared-libs/go/log"
)

type implUseCase struct {
	repo      repository.PostgresRepository
	searchUC  search.UseCase
	analytics analytics.Client
	llm       llm.LLM
	l         log.Logger
}

func New(
	repo repository.PostgresRepository,
	searchUC search.UseCase,
	analyticsClient analytics.Client,
	llmClient llm.LLM,
	l log.Logger,
) chat.UseCase {
	return &implUseCase{
		repo:      repo,
		searchUC:  searchUC,
		analytics: analyticsClient,
		llm:       llmClient,
		l:         l,
	}
}
