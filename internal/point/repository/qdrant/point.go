package qdrant

import (
	"context"
	"fmt"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
	pkgQdrant "knowledge-srv/pkg/qdrant"

	pb "github.com/qdrant/go-client/qdrant"
)

func (r *implRepository) Search(ctx context.Context, opt repository.SearchOptions) ([]point.SearchOutput, error) {
	pkgResults, err := r.client.SearchWithFilter(ctx, opt.CollectionName, opt.Vector, opt.Limit, opt.Filter, opt.ScoreThreshold)
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
	if err := r.client.UpsertPoints(ctx, opt.CollectionName, pkgPoints); err != nil {
		r.l.Errorf(ctx, "point.repository.qdrant.Upsert: Failed to upsert points: %v", err)
		return err
	}
	return nil
}

func (r *implRepository) Count(ctx context.Context, opt repository.CountOptions) (uint64, error) {
	return r.client.CountPoints(ctx, opt.CollectionName)
}

func (r *implRepository) Delete(ctx context.Context, opt repository.DeleteOptions) error {
	return fmt.Errorf("not implemented")
}

func (r *implRepository) Scroll(ctx context.Context, opt repository.ScrollOptions) ([]model.Point, error) {
	if opt.CollectionName == "" {
		return nil, fmt.Errorf("collection name is required")
	}
	maxTotal := opt.Limit
	if maxTotal == 0 {
		maxTotal = 1000
	}
	const batch = 100
	var all []model.Point
	var pbOffset *pb.PointId
	for uint64(len(all)) < maxTotal {
		n := batch
		if rem := int(maxTotal - uint64(len(all))); rem < n {
			n = rem
		}
		if n <= 0 {
			break
		}
		points, next, err := r.client.ScrollPoints(ctx, opt.CollectionName, opt.Filter, uint32(n), opt.WithPayload, pbOffset)
		if err != nil {
			r.l.Errorf(ctx, "point.repository.qdrant.Scroll: %v", err)
			return nil, err
		}
		for _, p := range points {
			all = append(all, model.Point{
				ID:      p.ID,
				Vector:  p.Vector,
				Payload: p.Payload,
			})
		}
		if next == nil || len(points) == 0 {
			break
		}
		pbOffset = next
	}
	if uint64(len(all)) > maxTotal {
		all = all[:maxTotal]
	}
	return all, nil
}
