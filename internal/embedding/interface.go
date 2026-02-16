package embedding

import (
	"context"
)

//go:generate mockery --name UseCase
type UseCase interface {
	Generate(ctx context.Context, input GenerateInput) (GenerateOutput, error)
	GenerateMany(ctx context.Context, input GenerateManyInput) (GenerateManyOutput, error)
}
