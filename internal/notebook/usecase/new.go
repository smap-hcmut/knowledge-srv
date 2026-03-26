package usecase

import (
	"sync"

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
