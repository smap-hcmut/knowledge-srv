package indexing

import (
	"context"
)

//go:generate mockery --name UseCase
type UseCase interface {
	IndexBatch(ctx context.Context, input IndexBatchInput) (IndexBatchOutput, error)
	IndexInsight(ctx context.Context, input IndexInsightInput) (IndexInsightOutput, error)
	IndexDigest(ctx context.Context, input IndexDigestInput) (IndexDigestOutput, error)
	RetryFailed(ctx context.Context, ip RetryFailedInput) (RetryFailedOutput, error)
	Reconcile(ctx context.Context, ip ReconcileInput) (ReconcileOutput, error)
	GetStatistics(ctx context.Context, projectID string) (StatisticOutput, error)
	PurgeProject(ctx context.Context, projectID string) (PurgeProjectOutput, error)
}

// PurgeProjectOutput reports what the cleanup pass removed so the caller can
// log/respond with how big the purge actually was.
type PurgeProjectOutput struct {
	ProjectID       string `json:"project_id"`
	CollectionName  string `json:"collection_name"`
	DocumentsDeleted int64  `json:"documents_deleted"`
}
