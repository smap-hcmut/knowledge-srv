package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"knowledge-srv/internal/indexing"
	kafkaDelivery "knowledge-srv/internal/indexing/delivery/kafka"
)

// PublishIndexingResult publishes an indexing result event
func (p *implProducer) PublishIndexingResult(ctx context.Context, result indexing.IndexingResult) error {
	// Convert to message DTO
	msg := kafkaDelivery.IndexingResultMessage{
		DocumentID:      result.DocumentID,
		KnowledgeBaseID: result.KnowledgeBaseID,
		Status:          result.Status,
		ChunksIndexed:   result.ChunksIndexed,
		ErrorMessage:    result.ErrorMessage,
		CompletedAt:     result.CompletedAt,
	}

	// Marshal to JSON
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal indexing result: %w", err)
	}

	// Publish to Kafka
	key := []byte(result.DocumentID)
	if err := p.producer.Publish(key, body); err != nil {
		return fmt.Errorf("failed to publish indexing result: %w", err)
	}

	p.l.Infof(ctx, "Published indexing result for document %s: %s", result.DocumentID, result.Status)
	return nil
}

// PublishIndexingProgress publishes an indexing progress event
func (p *implProducer) PublishIndexingProgress(ctx context.Context, progress indexing.IndexingProgress) error {
	// Convert to message DTO
	msg := kafkaDelivery.IndexingProgressMessage{
		DocumentID:      progress.DocumentID,
		KnowledgeBaseID: progress.KnowledgeBaseID,
		Progress:        progress.Progress,
		CurrentStep:     progress.CurrentStep,
		UpdatedAt:       progress.UpdatedAt,
	}

	// Marshal to JSON
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal indexing progress: %w", err)
	}

	// Publish to Kafka
	key := []byte(progress.DocumentID)
	if err := p.producer.Publish(key, body); err != nil {
		return fmt.Errorf("failed to publish indexing progress: %w", err)
	}

	p.l.Debugf(ctx, "Published indexing progress for document %s: %d%%", progress.DocumentID, progress.Progress)
	return nil
}
