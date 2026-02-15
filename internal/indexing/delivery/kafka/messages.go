package kafka

import (
	"time"
)

// BatchCompletedMessage - Kafka message cho analytics.batch.completed
type BatchCompletedMessage struct {
	BatchID     string    `json:"batch_id"`
	ProjectID   string    `json:"project_id"`
	FileURL     string    `json:"file_url"`
	RecordCount int       `json:"record_count"`
	CompletedAt time.Time `json:"completed_at"`
}
