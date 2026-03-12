package qdrant

import (
	"knowledge-srv/internal/point/repository"
	pkgQdrant "knowledge-srv/pkg/qdrant"

	"github.com/smap-hcmut/shared-libs/go/log"
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
