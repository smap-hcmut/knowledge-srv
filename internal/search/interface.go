package search

import (
	"context"
	"knowledge-srv/internal/model"
)

//go:generate mockery --name UseCase
type UseCase interface {
	Search(ctx context.Context, sc model.Scope, input SearchInput) (SearchOutput, error)
}
