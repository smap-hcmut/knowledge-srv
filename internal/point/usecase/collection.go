package usecase

import "context"

func (uc *implUseCase) EnsureCollection(ctx context.Context, name string, vectorSize uint64) error {
	return uc.repo.EnsureCollection(ctx, name, vectorSize)
}
