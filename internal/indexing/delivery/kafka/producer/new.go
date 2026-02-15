package producer

import (
	"knowledge-srv/internal/indexing"
	pkgKafka "knowledge-srv/pkg/kafka"
	"knowledge-srv/pkg/log"
)

// Producer interface for indexing domain
type Producer interface {
	indexing.Producer
}

// implProducer implements the Producer interface
type implProducer struct {
	l        log.Logger
	producer pkgKafka.IProducer
}

// New creates a new indexing producer
func New(l log.Logger, producer pkgKafka.IProducer) Producer {
	return &implProducer{
		l:        l,
		producer: producer,
	}
}
