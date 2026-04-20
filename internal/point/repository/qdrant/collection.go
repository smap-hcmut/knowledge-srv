package qdrant

import (
	"context"
	"fmt"

	pb "github.com/qdrant/go-client/qdrant"
)

// analyticsPayloadIndexes lists all fields that need payload indexes for faceting and filtering.
// keyword indexes are required for Facet queries; float index for range filters on timestamps.
var analyticsPayloadIndexes = []struct {
	field     string
	fieldType pb.FieldType
}{
	{"platform", pb.FieldType_FieldTypeKeyword},
	{"overall_sentiment", pb.FieldType_FieldTypeKeyword},
	{"risk_level", pb.FieldType_FieldTypeKeyword},
	{"aspects.aspect", pb.FieldType_FieldTypeKeyword},
	{"content_created_at", pb.FieldType_FieldTypeFloat},
}

func (r *implRepository) EnsureCollection(ctx context.Context, name string, vectorSize uint64) error {
	exists, err := r.client.CollectionExists(ctx, name)
	if err != nil {
		r.l.Errorf(ctx, "point.repository.qdrant.EnsureCollection: failed to check collection %s: %v", name, err)
		return err
	}

	if !exists {
		r.l.Infof(ctx, "point.repository.qdrant.EnsureCollection: creating collection %s (vectorSize=%d)", name, vectorSize)
		if err := r.client.CreateCollection(ctx, name, vectorSize, pb.Distance_Cosine); err != nil {
			r.l.Errorf(ctx, "point.repository.qdrant.EnsureCollection: failed to create collection %s: %v", name, err)
			return err
		}
		r.l.Infof(ctx, "point.repository.qdrant.EnsureCollection: collection %s created successfully", name)
	}

	// Always ensure payload indexes exist (idempotent — safe to call on existing indexes).
	if err := r.ensurePayloadIndexes(ctx, name); err != nil {
		return err
	}

	return nil
}

// ensurePayloadIndexes creates all required payload indexes for the collection.
// Qdrant silently accepts duplicate CreateFieldIndex calls, so this is idempotent.
func (r *implRepository) ensurePayloadIndexes(ctx context.Context, name string) error {
	for _, idx := range analyticsPayloadIndexes {
		if err := r.client.CreateFieldIndex(ctx, name, idx.field, idx.fieldType); err != nil {
			return fmt.Errorf("point.repository.qdrant.ensurePayloadIndexes: failed to create index %s on %s: %w", idx.field, name, err)
		}
	}
	r.l.Infof(ctx, "point.repository.qdrant.ensurePayloadIndexes: ensured %d payload indexes on %s", len(analyticsPayloadIndexes), name)
	return nil
}
