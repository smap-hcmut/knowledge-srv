package consumer

import (
	"context"
	"knowledge-srv/internal/indexing/delivery/kafka"
)

// ConsumeBatchCompleted starts consuming analytics.batch.completed messages
func (c *consumer) ConsumeBatchCompleted(ctx context.Context) error {
	// Create consumer group
	group, err := c.createConsumerGroup(kafka.GroupIDBatchCompleted)
	if err != nil {
		return err
	}
	c.batchCompletedGroup = group

	// Create handler
	handler := &batchCompletedHandler{
		consumer: c,
	}

	// Start consuming in goroutine with context
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := group.ConsumeWithContext(ctx, []string{kafka.TopicBatchCompleted}, handler); err != nil {
					c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.ConsumeBatchCompleted: Consumer error: %v", err)
				}
			}
		}
	}()

	// Start error handler
	go func() {
		for err := range group.Errors() {
			c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.ConsumeBatchCompleted: Consumer group error: %v", err)
		}
	}()

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.ConsumeBatchCompleted: Started consuming topic: %s (group: %s)", kafka.TopicBatchCompleted, kafka.GroupIDBatchCompleted)

	return nil
}
