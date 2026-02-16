package usecase

import (
	"context"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
)

func (uc *implUseCase) Search(ctx context.Context, input point.SearchInput) ([]point.SearchOutput, error) {
	return uc.repo.Search(ctx, repository.SearchOptions{
		Vector:         input.Vector,
		Filter:         input.Filter,
		Limit:          input.Limit,
		WithPayload:    input.WithPayload,
		ScoreThreshold: input.ScoreThreshold,
	})
}
