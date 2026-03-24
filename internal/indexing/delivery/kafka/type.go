package kafka

import (
	"time"
)

const (
	TopicBatchCompleted   = "analytics.batch.completed"
	GroupIDBatchCompleted = "knowledge-indexing-batch"
)

// BatchCompletedMessage - Kafka message cho analytics.batch.completed
type BatchCompletedMessage struct {
	BatchID     string    `json:"batch_id"`
	ProjectID   string    `json:"project_id"`
	CampaignID  string    `json:"campaign_id,omitempty"` // Optional: for notebook sync. Fallback: resolve via project-srv.
	FileURL     string    `json:"file_url"`
	RecordCount int       `json:"record_count"`
	CompletedAt time.Time `json:"completed_at"`
}
