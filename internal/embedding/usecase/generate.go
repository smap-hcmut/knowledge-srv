package usecase

import (
	"context"
	"crypto/sha256"
	"fmt"
	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/embedding/repository"
)

func (uc *implUseCase) Generate(ctx context.Context, input embedding.GenerateInput) (embedding.GenerateOutput, error) {
	if input.Text == "" {
		uc.l.Errorf(ctx, "embedding.usecase.Generate: empty text")
		return embedding.GenerateOutput{}, embedding.ErrEmptyText
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(input.Text)))

	// 1. Check cache
	cached, err := uc.repo.Get(ctx, repository.GetOptions{Key: hash})
	if err == nil && cached != nil {
		uc.l.Debugf(ctx, "embedding.usecase.Generate: cache hit")
		return embedding.GenerateOutput{Vector: cached}, nil
	}

	// 2. Call Voyage
	vectors, err := uc.voyage.Embed(ctx, []string{input.Text})
	if err != nil {
		uc.l.Errorf(ctx, "embedding.usecase.Generate: Voyage embed failed: %v", err)
		return embedding.GenerateOutput{}, err
	}
	if len(vectors) == 0 {
		uc.l.Errorf(ctx, "embedding.usecase.Generate: no vector returned")
		return embedding.GenerateOutput{}, embedding.ErrNoVectorReturned
	}
	vector := vectors[0]

	// 3. Save cache
	if err := uc.repo.Save(ctx, repository.SaveOptions{
		Key:    hash,
		Vector: vector,
	}); err != nil {
		uc.l.Errorf(ctx, "embedding.usecase.Generate: cache save failed: %v", err)
		return embedding.GenerateOutput{}, err
	}

	return embedding.GenerateOutput{Vector: vector}, nil
}

func (uc *implUseCase) GenerateMany(ctx context.Context, input embedding.GenerateManyInput) (embedding.GenerateManyOutput, error) {
	if len(input.Texts) == 0 {
		uc.l.Errorf(ctx, "embedding.usecase.GenerateMany: empty texts")
		return embedding.GenerateManyOutput{}, embedding.ErrEmptyTexts
	}
	results := make([][]float32, len(input.Texts))
	hashes := make([]string, len(input.Texts))
	missIndices := []int{}
	missTexts := []string{}

	// 1. Check cache for each
	for i, text := range input.Texts {
		hashes[i] = fmt.Sprintf("%x", sha256.Sum256([]byte(text)))
		cached, err := uc.repo.Get(ctx, repository.GetOptions{Key: hashes[i]})
		if err == nil && cached != nil {
			results[i] = cached
		} else {
			missIndices = append(missIndices, i)
			missTexts = append(missTexts, text)
		}
	}

	if len(missIndices) == 0 {
		return embedding.GenerateManyOutput{Vectors: results}, nil
	}

	// 2. Call Voyage for misses
	vectors, err := uc.voyage.Embed(ctx, missTexts)
	if err != nil {
		uc.l.Errorf(ctx, "embedding.usecase.GenerateMany: Voyage embed failed: %v", err)
		return embedding.GenerateManyOutput{}, err
	}
	if len(vectors) != len(missTexts) {
		uc.l.Errorf(ctx, "embedding.usecase.GenerateMany: mismatch vector count")
		return embedding.GenerateManyOutput{}, embedding.ErrMismatchVectorCount
	}

	// 3. Save cache for misses and fill results
	for i, vector := range vectors {
		origIdx := missIndices[i]
		results[origIdx] = vector
		if err := uc.repo.Save(ctx, repository.SaveOptions{
			Key:    hashes[origIdx],
			Vector: vector,
		}); err != nil {
			uc.l.Errorf(ctx, "embedding.usecase.GenerateMany: cache save failed: %v", err)
			return embedding.GenerateManyOutput{}, err
		}
	}

	return embedding.GenerateManyOutput{Vectors: results}, nil
}
