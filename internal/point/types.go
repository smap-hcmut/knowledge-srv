package point

import (
	"knowledge-srv/internal/model"

	"github.com/qdrant/go-client/qdrant"
)

type Filter = qdrant.Filter

type SearchInput struct {
	CollectionName string
	Vector         []float32
	Filter         *Filter
	Limit          uint64
	WithPayload    bool
	ScoreThreshold float32
}

type SearchOutput struct {
	ID      string
	Score   float32
	Payload map[string]interface{}
}

type UpsertInput struct {
	CollectionName string
	Points         []model.Point
}

type CountInput struct {
	CollectionName string
	Filter         *Filter
}

type DeleteInput struct {
	CollectionName string
	Filter         *Filter
	Points         []string
}

type ScrollInput struct {
	CollectionName string
	Filter         *Filter
	Limit          uint64
	WithPayload    bool
	Offset         *string
}

type FacetInput struct {
	CollectionName string
	Key            string
	Filter         *Filter
	Limit          uint64
}

type FacetOutput struct {
	Value string
	Count uint64
}
