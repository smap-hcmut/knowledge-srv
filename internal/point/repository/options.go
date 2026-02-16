package repository

import (
	"knowledge-srv/internal/model"

	"github.com/qdrant/go-client/qdrant"
)

type SearchOptions struct {
	Vector         []float32
	Filter         *qdrant.Filter
	Limit          uint64
	WithPayload    bool
	ScoreThreshold float32
}

type UpsertOptions struct {
	Points []model.Point
}

type CountOptions struct {
	Filter *qdrant.Filter
}

type DeleteOptions struct {
	Filter *qdrant.Filter
	Points []string
}

type ScrollOptions struct {
	Filter      *qdrant.Filter
	Limit       uint64
	WithPayload bool
	Offset      *string
}
