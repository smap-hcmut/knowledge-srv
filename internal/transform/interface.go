package transform

import (
	"context"

	"knowledge-srv/internal/notebook"
)

//go:generate mockery --name UseCase
type UseCase interface {
	// BuildParts loads Layer 1–3 content from Qdrant and assembles markdown parts for NotebookLM.
	BuildParts(ctx context.Context, input TransformInput) ([]notebook.MarkdownPart, error)
}
