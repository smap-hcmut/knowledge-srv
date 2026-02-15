package usecase

import (
	"context"
	"time"

	"knowledge-srv/internal/indexing"
	repo "knowledge-srv/internal/indexing/repository"
)

// RetryFailed - Retry các records FAILED
func (uc *implUseCase) RetryFailed(
	ctx context.Context,
	input indexing.RetryFailedInput,
) (indexing.RetryFailedOutput, error) {
	startTime := time.Now()

	// Step 1: Query failed records từ DB
	docs, err := uc.repo.ListDocuments(ctx, repo.ListDocumentsOptions{
		Status:   "FAILED",
		MaxRetry: input.MaxRetryCount,
		Limit:    input.Limit,
		OrderBy:  "created_at ASC",
	})
	if err != nil {
		return indexing.RetryFailedOutput{}, err
	}

	// Step 2: Retry each record
	var succeeded, failed int
	for _, doc := range docs {
		// Increment retry count
		newRetryCount := doc.RetryCount + 1

		// TODO: Implement actual retry logic
		// For now, just mark as pending for re-processing
		_, _ = uc.repo.UpdateDocumentStatus(ctx, repo.UpdateDocumentStatusOptions{
			ID:     doc.ID,
			Status: "PENDING",
			Metrics: repo.DocumentStatusMetrics{
				RetryCount: newRetryCount,
			},
		})
	}

	return indexing.RetryFailedOutput{
		TotalRetried: len(docs),
		Succeeded:    succeeded,
		Failed:       failed,
		Duration:     time.Since(startTime),
	}, nil
}
