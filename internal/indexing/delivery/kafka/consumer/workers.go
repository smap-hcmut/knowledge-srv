package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IBM/sarama"

	kafkaDelivery "knowledge-srv/internal/indexing/delivery/kafka"
	"knowledge-srv/internal/model"
	"knowledge-srv/pkg/scope"
)

// handleBatchCompletedMessage receives message, normalizes scope + input, delegates to usecase (no business logic here).
func (c *consumer) handleBatchCompletedMessage(msg *sarama.ConsumerMessage) error {
	ctx := context.Background()

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: Processing message from partition %d, offset %d",
		msg.Partition, msg.Offset)

	// 1. Unmarshal message
	var message kafkaDelivery.BatchCompletedMessage
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		c.l.Warnf(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: Invalid message format (skipping): %v", err)
		return nil // Skip invalid messages
	}

	// 2. Validate message (format only; business rules stay in usecase)
	if message.BatchID == "" || message.FileURL == "" {
		c.l.Warnf(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: Invalid message: missing required fields (skipping)")
		return nil
	}

	// 3. Map to usecase input (presenter)
	input := toIndexInput(message)

	// 4. Create scope (system scope for background processing) and set to context
	sc := model.Scope{
		UserID: "system",
		Role:   "system",
	}
	ctx = scope.SetScopeToContext(ctx, sc)

	// 5. Call UseCase (scope already in context)
	output, err := c.uc.Index(ctx, input)
	if err != nil {
		c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: usecase Index failed: %v", err)
		return fmt.Errorf("usecase error: %w", err)
	}

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: Successfully processed batch %s: indexed=%d, failed=%d, skipped=%d",
		message.BatchID, output.Indexed, output.Failed, output.Skipped)
	return nil
}
