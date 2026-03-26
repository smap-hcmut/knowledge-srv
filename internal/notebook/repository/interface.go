package repository

import (
	"context"

	"knowledge-srv/internal/notebook"
)

// CampaignRepo maps campaigns to NotebookLM notebooks (one row per campaign + period).
type CampaignRepo interface {
	GetByCampaignAndPeriod(ctx context.Context, campaignID, periodLabel string) (notebook.NotebookInfo, error)
	Create(ctx context.Context, campaignID, periodLabel, notebookID string) error
}

// SourceRepo tracks markdown source uploads for deduplication and webhook correlation.
type SourceRepo interface {
	GetByContentHash(ctx context.Context, campaignID, contentHash string) (notebook.SourceRecord, error)
	CreateUploading(ctx context.Context, in notebook.SourceUpsertInput) error
	ListFailedRetryable(ctx context.Context, maxRetries int) ([]notebook.SourceRecord, error)
	UpdateStatusByMaestroJobID(ctx context.Context, maestroJobID, status string, errMsg *string) error
	HasSyncedForCampaign(ctx context.Context, campaignID string) (bool, error)
}

// SessionRepo reserved for persisting Maestro browser sessions (optional).
type SessionRepo interface{}

// ChatJobRepo persists async notebook chat jobs.
type ChatJobRepo interface {
	CreateJob(ctx context.Context, job notebook.ChatJob) (notebook.ChatJob, error)
	GetJobByID(ctx context.Context, jobID string) (notebook.ChatJob, error)
	UpdateJobStatus(ctx context.Context, jobID, status string, maestroJobID, answer *string, fallbackUsed bool) error
}
