package usecase

import (
	"sync"
	"time"

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

	suggestionCache   map[string]suggestionCacheEntry
	suggestionCacheMu sync.RWMutex
}

// suggestionCacheEntry memoises a /suggestions response. The endpoint
// fans out 6 Qdrant requests per project (count + 2 sentiment + platform +
// 2 aspect) so a campaign with 4 projects runs 24 parallel calls per
// hit — p99 was ~487 ms. A 60 s campaign-scoped cache drops repeat hits
// to a map lookup.
type suggestionCacheEntry struct {
	response  chat.SuggestionOutput
	expiresAt time.Time
}

const suggestionCacheTTL = 60 * time.Second

func New(
	repo repository.PostgresRepository,
	searchUC search.UseCase,
	analyticsClient analytics.Client,
	llmClient llm.LLM,
	l log.Logger,
) chat.UseCase {
	return &implUseCase{
		repo:            repo,
		searchUC:        searchUC,
		analytics:       analyticsClient,
		llm:             llmClient,
		l:               l,
		suggestionCache: make(map[string]suggestionCacheEntry),
	}
}
