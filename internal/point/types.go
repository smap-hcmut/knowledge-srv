package point

import (
	"knowledge-srv/internal/model"

	"github.com/qdrant/go-client/qdrant"
)

type Filter = qdrant.Filter

type SearchInput struct {
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
	Points []model.Point
}

type CountInput struct {
	Filter *Filter
}

type DeleteInput struct {
	Filter *Filter
	Points []string
}

type ScrollInput struct {
	Filter      *Filter
	Limit       uint64
	WithPayload bool
	Offset      *string
}
