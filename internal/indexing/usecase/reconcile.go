package usecase

import (
	"context"
	"time"

	"knowledge-srv/internal/indexing"
	repo "knowledge-srv/internal/indexing/repository"
)

// Reconcile - Reconcile records PENDING quá lâu
func (uc *implUseCase) Reconcile(
	ctx context.Context,
	input indexing.ReconcileInput,
) (indexing.ReconcileOutput, error) {
	startTime := time.Now()

	// Step 1: Find stale PENDING records
	staleBefore := time.Now().Add(-input.StaleDuration)
	docs, err := uc.repo.ListDocuments(ctx, repo.ListDocumentsOptions{
		Status:      "PENDING",
		StaleBefore: &staleBefore,
		Limit:       input.Limit,
		OrderBy:     "created_at ASC",
	})
	if err != nil {
		return indexing.ReconcileOutput{}, err
	}

	// Step 2: Reconcile each record
	var fixed, requeued int
	for _, doc := range docs {
		// For now, mark as FAILED to trigger retry
		_, _ = uc.repo.UpdateDocumentStatus(ctx, repo.UpdateDocumentStatusOptions{
			ID:     doc.ID,
			Status: "FAILED",
			Metrics: repo.DocumentStatusMetrics{
				ErrorMessage: "Reconciliation: PENDING timeout",
			},
		})
		requeued++
	}

	return indexing.ReconcileOutput{
		TotalChecked: len(docs),
		Fixed:        fixed,
		Requeued:     requeued,
		Duration:     time.Since(startTime),
	}, nil
}
