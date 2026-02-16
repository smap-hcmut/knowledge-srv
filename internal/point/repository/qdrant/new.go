package qdrant

import (
	"knowledge-srv/internal/point/repository"
	"knowledge-srv/pkg/log"
	pkgQdrant "knowledge-srv/pkg/qdrant"
)

type implRepository struct {
	client pkgQdrant.IQdrant
	l      log.Logger
}

func New(client pkgQdrant.IQdrant, l log.Logger) repository.QdrantRepository {
	return &implRepository{
		client: client,
		l:      l,
	}
}
