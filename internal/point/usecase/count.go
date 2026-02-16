package usecase

import (
	"context"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
)

func (uc *implUseCase) Count(ctx context.Context, input point.CountInput) (uint64, error) {
	return uc.repo.Count(ctx, repository.CountOptions{
		Filter: input.Filter,
	})
}
