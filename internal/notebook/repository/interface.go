package repository

import (
	"context"

	"knowledge-srv/internal/notebook"
)

// CampaignRepo defines the interface for campaign repository operations.
type CampaignRepo interface {
	// Placeholder for campaign repository methods
}

// SourceRepo defines the interface for source repository operations.
type SourceRepo interface {
	// Placeholder for source repository methods
}

// SessionRepo defines the interface for session repository operations.
type SessionRepo interface {
	// Placeholder for session repository methods
}

// ChatJobRepo defines the interface for chat job repository operations.
type ChatJobRepo interface {
	CreateJob(ctx context.Context, job notebook.ChatJob) (notebook.ChatJob, error)
	GetJobByID(ctx context.Context, jobID string) (notebook.ChatJob, error)
	UpdateJobStatus(ctx context.Context, jobID, status string, maestroJobID, answer *string, fallbackUsed bool) error
}

