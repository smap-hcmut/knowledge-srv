package notebook

import (
	"context"

	"knowledge-srv/internal/model"
)

// UseCase defines the interface for the notebook domain.
type UseCase interface {
	// StartSessionLoop starts the session lifecycle management loop.
	StartSessionLoop(ctx context.Context, sc model.Scope) error

	// StopSessionLoop stops the session lifecycle management loop.
	StopSessionLoop(ctx context.Context, sc model.Scope) error

	// EnsureNotebook ensures that a notebook exists for the given campaign and period.
	EnsureNotebook(ctx context.Context, sc model.Scope, campaignID, periodLabel string) (NotebookInfo, error)

	// SyncPart synchronizes a part of the notebook with the given input.
	SyncPart(ctx context.Context, sc model.Scope, input SyncPartInput) error

	// RetryFailed retries any failed notebook synchronization jobs.
	RetryFailed(ctx context.Context, sc model.Scope) (RetryOutput, error)

	// HandleWebhook processes webhook callbacks from Maestro.
	HandleWebhook(ctx context.Context, sc model.Scope, payload WebhookPayload) error

	// SubmitChatJob submits a chat message to a notebook and returns a job ID asynchronously.
	SubmitChatJob(ctx context.Context, sc model.Scope, conversationID, campaignID, userMessage string) (string, error)

	// GetChatJobStatus retrieves the status and result of a chat job.
	GetChatJobStatus(ctx context.Context, sc model.Scope, jobID string) (ChatJob, error)

	// HasSyncedForCampaign is true when at least one NotebookLM source finished syncing for this campaign.
	HasSyncedForCampaign(ctx context.Context, campaignID string) (bool, error)

	// ApplyChatFallback completes a chat job with a Qdrant/Gemini answer when NotebookLM times out or is unavailable.
	ApplyChatFallback(ctx context.Context, jobID, answer string) error
}
