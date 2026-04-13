package usecase

import (
	"knowledge-srv/internal/chat"
	"knowledge-srv/internal/chat/repository"
	"knowledge-srv/internal/notebook"
	"knowledge-srv/internal/search"

	"github.com/smap-hcmut/shared-libs/go/llm"
	"github.com/smap-hcmut/shared-libs/go/log"
)

type implUseCase struct {
	repo       repository.PostgresRepository
	searchUC   search.UseCase
	notebookUC notebook.UseCase
	llm        llm.LLM
	cfg        Config
	l          log.Logger
}

func New(
	repo repository.PostgresRepository,
	searchUC search.UseCase,
	notebookUC notebook.UseCase,
	llmClient llm.LLM,
	cfg Config,
	l log.Logger,
) chat.UseCase {
	return &implUseCase{
		repo:       repo,
		searchUC:   searchUC,
		notebookUC: notebookUC,
		llm:        llmClient,
		cfg:        cfg,
		l:          l,
	}
}
