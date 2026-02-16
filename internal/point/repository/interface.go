package repository

import (
	"context"

	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
)

//go:generate mockery --name QdrantRepository
type QdrantRepository interface {
	Search(ctx context.Context, opt SearchOptions) ([]point.SearchOutput, error)
	Upsert(ctx context.Context, opt UpsertOptions) error
	Count(ctx context.Context, opt CountOptions) (uint64, error)
	Delete(ctx context.Context, opt DeleteOptions) error
	Scroll(ctx context.Context, opt ScrollOptions) ([]model.Point, error)
}
