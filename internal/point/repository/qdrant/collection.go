package qdrant

import (
	"context"

	pb "github.com/qdrant/go-client/qdrant"
)

func (r *implRepository) EnsureCollection(ctx context.Context, name string, vectorSize uint64) error {
	exists, err := r.client.CollectionExists(ctx, name)
	if err != nil {
		r.l.Errorf(ctx, "point.repository.qdrant.EnsureCollection: failed to check collection %s: %v", name, err)
		return err
	}
	if exists {
		return nil
	}

	r.l.Infof(ctx, "point.repository.qdrant.EnsureCollection: creating collection %s (vectorSize=%d)", name, vectorSize)
	if err := r.client.CreateCollection(ctx, name, vectorSize, pb.Distance_Cosine); err != nil {
		r.l.Errorf(ctx, "point.repository.qdrant.EnsureCollection: failed to create collection %s: %v", name, err)
		return err
	}

	r.l.Infof(ctx, "point.repository.qdrant.EnsureCollection: collection %s created successfully", name)
	return nil
}
