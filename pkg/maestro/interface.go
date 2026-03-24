package maestro

import (
	"context"
	"fmt"

	pkghttp "github.com/smap-hcmut/shared-libs/go/httpclient"
)

// IMaestro defines the interface for Maestro NotebookLM automation API.
// Pure I/O client — no state, no session management, no business logic.
// Implementations are safe for concurrent use.
type IMaestro interface {
	// Sessions
	CreateSession(ctx context.Context, req CreateSessionReq) (SessionData, error)
	GetSession(ctx context.Context, sessionID string) (SessionData, error)
	DeleteSession(ctx context.Context, sessionID string) error

	// Notebooks (require sessionID)
	CreateNotebook(ctx context.Context, sessionID string, req CreateNotebookReq) (JobEnqueued, error)
	ListNotebooks(ctx context.Context, sessionID string) ([]NotebookData, error)
	UploadSources(ctx context.Context, sessionID, notebookID string, req UploadSourcesReq) (JobEnqueued, error)
	ChatNotebook(ctx context.Context, sessionID, notebookID string, req ChatNotebookReq) (JobEnqueued, error)

	// Jobs
	GetJob(ctx context.Context, jobID string) (JobData, error)

	// Pipelines
	SubmitPipeline(ctx context.Context, sessionID string, req PipelineReq) (PipelineData, error)
}

// NewMaestro creates a new Maestro client. BaseURL and APIKey must be set.
func NewMaestro(cfg MaestroConfig) (IMaestro, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("maestro: base URL is required")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("maestro: API key is required")
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = pkghttp.NewClient(pkghttp.Config{
			Timeout:   DefaultTimeout,
			Retries:   DefaultRetries,
			RetryWait: DefaultRetryWait,
		})
	}

	return &maestroImpl{
		baseURL:    cfg.BaseURL,
		apiKey:     cfg.APIKey,
		httpClient: httpClient,
	}, nil
}
