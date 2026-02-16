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
	docs, err := uc.postgreRepo.ListDocuments(ctx, repo.ListDocumentsOptions{
		Status:     indexing.STATUS_FAILED,
		MaxRetry:   input.MaxRetryCount,
		ErrorTypes: input.ErrorTypes,
		Limit:      input.Limit,
		OrderBy:    "created_at ASC",
	})
	if err != nil {
		return indexing.RetryFailedOutput{}, err
	}

	// Step 2: Retry each record
	var succeeded, failed int
	for _, doc := range docs {
		newRetryCount := doc.RetryCount + 1

		// Mark as PENDING for re-processing with incremented retry count
		_, updateErr := uc.postgreRepo.UpdateDocumentStatus(ctx, repo.UpdateDocumentStatusOptions{
			ID:     doc.ID,
			Status: indexing.STATUS_PENDING,
			Metrics: repo.DocumentStatusMetrics{
				RetryCount:   newRetryCount,
				ErrorMessage: "", // Clear previous error
			},
		})
		if updateErr != nil {
			uc.l.Warnf(ctx, "indexing.usecase.RetryFailed: Failed to update doc %s: %v", doc.ID, updateErr)
			failed++
			continue
		}

		// Resolve matching DLQ entry
		dlqEntry, _ := uc.postgreRepo.GetOneDLQ(ctx, repo.GetOneDLQOptions{
			AnalyticsID: doc.AnalyticsID,
		})
		if dlqEntry.ID != "" {
			_ = uc.postgreRepo.MarkResolvedDLQ(ctx, dlqEntry.ID)
		}

		succeeded++
	}

	return indexing.RetryFailedOutput{
		TotalRetried: len(docs),
		Succeeded:    succeeded,
		Failed:       failed,
		Duration:     time.Since(startTime),
	}, nil
}
