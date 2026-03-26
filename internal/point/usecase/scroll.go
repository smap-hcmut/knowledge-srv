package usecase

import (
	"context"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
)

func (uc *implUseCase) Scroll(ctx context.Context, input point.ScrollInput) ([]model.Point, error) {
	return uc.repo.Scroll(ctx, repository.ScrollOptions{
		CollectionName: input.CollectionName,
		Filter:         input.Filter,
		Limit:          input.Limit,
		WithPayload:    input.WithPayload,
		Offset:         input.Offset,
	})
}
