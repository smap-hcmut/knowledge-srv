package usecase

import (
	"context"
	"errors"
	"sync"

	"knowledge-srv/internal/model"
	"knowledge-srv/internal/notebook"
	"knowledge-srv/internal/notebook/repository"
	"knowledge-srv/pkg/maestro"

	"github.com/smap-hcmut/shared-libs/go/log"
)

// implUseCase is the implementation of the notebook.UseCase interface.
type implUseCase struct {
	maestro       maestro.IMaestro
	campaignRepo  repository.CampaignRepo
	sourceRepo    repository.SourceRepo
	sessionRepo   repository.SessionRepo
	chatJobRepo   repository.ChatJobRepo
	cfg           Config
	l             log.Logger
	mu            sync.RWMutex
	activeSession string
}

// NewUseCase creates a new instance of the notebook usecase.
func NewUseCase(
	maestro maestro.IMaestro,
	campaignRepo repository.CampaignRepo,
	sourceRepo repository.SourceRepo,
	sessionRepo repository.SessionRepo,
	chatJobRepo repository.ChatJobRepo,
	cfg Config,
	l log.Logger,
) notebook.UseCase {
	return &implUseCase{
		maestro:      maestro,
		campaignRepo: campaignRepo,
		sourceRepo:   sourceRepo,
		sessionRepo:  sessionRepo,
		chatJobRepo:  chatJobRepo,
		cfg:          cfg,
		l:            l,
	}
}

// --- Stubs for unimplemented interface methods ---

func (uc *implUseCase) EnsureNotebook(ctx context.Context, sc model.Scope, campaignID, periodLabel string) (notebook.NotebookInfo, error) {
	return notebook.NotebookInfo{}, errors.New("not implemented")
}

func (uc *implUseCase) SyncPart(ctx context.Context, sc model.Scope, input notebook.SyncPartInput) error {
	return errors.New("not implemented")
}

func (uc *implUseCase) RetryFailed(ctx context.Context, sc model.Scope) (notebook.RetryOutput, error) {
	return notebook.RetryOutput{}, errors.New("not implemented")
}


