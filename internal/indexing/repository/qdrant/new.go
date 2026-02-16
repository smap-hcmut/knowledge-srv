package qdrant

import (
	repo "knowledge-srv/internal/indexing/repository"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/qdrant"
)

type implQdrantRepository struct {
	qdrant qdrant.IQdrant
	l      log.Logger
}

func New(qdrant qdrant.IQdrant, l log.Logger) repo.QdrantRepository {
	return &implQdrantRepository{
		qdrant: qdrant,
		l:      l,
	}
}
