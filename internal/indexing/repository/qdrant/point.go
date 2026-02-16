package qdrant

import (
	"context"

	repo "knowledge-srv/internal/indexing/repository"
	"knowledge-srv/pkg/qdrant"
)

const collectionName = "knowledge_indexing"

func (r *implQdrantRepository) UpsertPoint(ctx context.Context, opt repo.UpsertPointOptions) error {
	point := qdrant.Point{
		ID:      opt.PointID,
		Vector:  opt.Vector,
		Payload: opt.Payload,
	}

	if err := r.qdrant.UpsertPoints(ctx, collectionName, []qdrant.Point{point}); err != nil {
		r.l.Errorf(ctx, "indexing.repository.qdrant.UpsertPoint: Failed to upsert point: %v", err)
		return repo.ErrFailedToUpsert
	}

	return nil
}
