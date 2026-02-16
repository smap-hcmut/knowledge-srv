package qdrant

import (
	"context"
	"fmt"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
	pkgQdrant "knowledge-srv/pkg/qdrant"
)

const collectionName = "smap_analytics"

func (r *implRepository) Search(ctx context.Context, opt repository.SearchOptions) ([]point.SearchOutput, error) {
	pkgResults, err := r.client.SearchWithFilter(ctx, collectionName, opt.Vector, opt.Limit, opt.Filter)
	if err != nil {
		r.l.Errorf(ctx, "point.repository.qdrant.Search: Failed to search points: %v", err)
		return nil, err
	}

	results := make([]point.SearchOutput, len(pkgResults))
	for i, pr := range pkgResults {
		results[i] = point.SearchOutput{
			ID:      pr.ID,
			Score:   pr.Score,
			Payload: pr.Payload,
		}
	}
	return results, nil
}

func (r *implRepository) Upsert(ctx context.Context, opt repository.UpsertOptions) error {
	pkgPoints := make([]pkgQdrant.Point, len(opt.Points))
	for i, p := range opt.Points {
		pkgPoints[i] = pkgQdrant.Point{
			ID:      p.ID,
			Vector:  p.Vector,
			Payload: p.Payload,
		}
	}
	if err := r.client.UpsertPoints(ctx, collectionName, pkgPoints); err != nil {
		r.l.Errorf(ctx, "point.repository.qdrant.Upsert: Failed to upsert points: %v", err)
		return err
	}
	return nil
}

func (r *implRepository) Count(ctx context.Context, opt repository.CountOptions) (uint64, error) {
	return r.client.CountPoints(ctx, collectionName)
}

func (r *implRepository) Delete(ctx context.Context, opt repository.DeleteOptions) error {
	return fmt.Errorf("not implemented")
}

func (r *implRepository) Scroll(ctx context.Context, opt repository.ScrollOptions) ([]model.Point, error) {
	return nil, fmt.Errorf("not implemented")
}
