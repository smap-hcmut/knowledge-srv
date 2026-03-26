package usecase

import (
	"context"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
)

func (uc *implUseCase) Upsert(ctx context.Context, input point.UpsertInput) error {
	return uc.repo.Upsert(ctx, repository.UpsertOptions{
		CollectionName: input.CollectionName,
		Points:         input.Points,
	})
}
