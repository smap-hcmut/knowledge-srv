package indexing

import (
	"context"
)

//go:generate mockery --name UseCase
type UseCase interface {
	Index(ctx context.Context, input IndexInput) (IndexOutput, error)
	IndexBatch(ctx context.Context, input IndexBatchInput) (IndexBatchOutput, error)
	IndexInsight(ctx context.Context, input IndexInsightInput) (IndexInsightOutput, error)
	IndexDigest(ctx context.Context, input IndexDigestInput) (IndexDigestOutput, error)
	RetryFailed(ctx context.Context, ip RetryFailedInput) (RetryFailedOutput, error)
	Reconcile(ctx context.Context, ip ReconcileInput) (ReconcileOutput, error)
	GetStatistics(ctx context.Context, projectID string) (StatisticOutput, error)
}
