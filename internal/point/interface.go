package point

import (
	"context"
	"knowledge-srv/internal/model"
)

//go:generate mockery --name UseCase
type UseCase interface {
	Search(ctx context.Context, input SearchInput) ([]SearchOutput, error)
	Upsert(ctx context.Context, input UpsertInput) error
	Count(ctx context.Context, input CountInput) (uint64, error)
	Delete(ctx context.Context, input DeleteInput) error
	Scroll(ctx context.Context, input ScrollInput) ([]model.Point, error)
	Facet(ctx context.Context, input FacetInput) ([]FacetOutput, error)
}
