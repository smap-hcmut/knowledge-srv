package consumer

import (
	"knowledge-srv/internal/indexing"
	kafkaDelivery "knowledge-srv/internal/indexing/delivery/kafka"
)

// toIndexInput maps Kafka message DTO to usecase input (delivery → usecase boundary).
func toIndexInput(m kafkaDelivery.BatchCompletedMessage) indexing.IndexInput {
	return indexing.IndexInput{
		BatchID:     m.BatchID,
		ProjectID:   m.ProjectID,
		FileURL:     m.FileURL,
		RecordCount: m.RecordCount,
	}
}
