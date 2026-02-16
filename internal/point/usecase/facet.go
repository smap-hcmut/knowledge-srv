package usecase

import (
	"context"

	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
)

func (uc *implUseCase) Facet(ctx context.Context, input point.FacetInput) ([]point.FacetOutput, error) {
	return uc.repo.Facet(ctx, repository.FacetOptions{
		Key:    input.Key,
		Filter: input.Filter,
		Limit:  input.Limit,
	})
}
