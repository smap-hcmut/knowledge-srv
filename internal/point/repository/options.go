package repository

import (
	"knowledge-srv/internal/model"

	"github.com/qdrant/go-client/qdrant"
)

type SearchOptions struct {
	CollectionName string
	Vector         []float32
	Filter         *qdrant.Filter
	Limit          uint64
	WithPayload    bool
	ScoreThreshold float32
}

type UpsertOptions struct {
	CollectionName string
	Points         []model.Point
}

type CountOptions struct {
	CollectionName string
	Filter         *qdrant.Filter
}

type DeleteOptions struct {
	CollectionName string
	Filter         *qdrant.Filter
	Points         []string
}

type ScrollOptions struct {
	CollectionName string
	Filter         *qdrant.Filter
	Limit          uint64
	WithPayload    bool
	Offset         *string
}

type FacetOptions struct {
	CollectionName string
	Key            string
	Filter         *qdrant.Filter
	Limit          uint64
}
