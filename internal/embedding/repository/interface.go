package repository

import "context"

//go:generate mockery --name Repository
type Repository interface {
	Get(ctx context.Context, opt GetOptions) ([]float32, error)
	Save(ctx context.Context, opt SaveOptions) error
}
