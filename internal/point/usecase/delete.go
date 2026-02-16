package usecase

import (
	"context"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
)

func (uc *implUseCase) Delete(ctx context.Context, input point.DeleteInput) error {
	return uc.repo.Delete(ctx, repository.DeleteOptions{
		Filter: input.Filter,
		Points: input.Points,
	})
}
