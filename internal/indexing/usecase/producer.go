package usecase

import (
	"context"
	"knowledge-srv/internal/indexing"
)

// PublishIndexingResult publishes an indexing result
func (uc *implUseCase) PublishIndexingResult(ctx context.Context, result indexing.IndexingResult) error {
	return uc.producer.PublishIndexingResult(ctx, result)
}

// PublishIndexingProgress publishes indexing progress
func (uc *implUseCase) PublishIndexingProgress(ctx context.Context, progress indexing.IndexingProgress) error {
	return uc.producer.PublishIndexingProgress(ctx, progress)
}
