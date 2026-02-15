package consumer

import (
	"context"
)

// ConsumeBatchCompleted starts consuming analytics.batch.completed messages
func (c *Consumer) ConsumeBatchCompleted(ctx context.Context, topic string) error {
	// Create consumer group
	groupID := "knowledge-indexing-batch"
	group, err := c.createConsumerGroup(groupID)
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
				if err := group.ConsumeWithContext(ctx, []string{topic}, handler); err != nil {
					c.l.Errorf(ctx, "Consumer error: %v", err)
				}
			}
		}
	}()

	// Start error handler
	go func() {
		for err := range group.Errors() {
			c.l.Errorf(ctx, "Consumer group error: %v", err)
		}
	}()

	c.l.Infof(ctx, "Started consuming topic: %s (group: %s)", topic, groupID)

	return nil
}
