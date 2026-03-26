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

// ConsumeReportDigest starts consuming analytics.report.digest messages
func (c *consumer) ConsumeReportDigest(ctx context.Context) error {
	group, err := c.createConsumerGroup(kafka.GroupIDReportDigest)
	if err != nil {
		return err
	}
	c.reportDigestGroup = group

	handler := &reportDigestHandler{consumer: c}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := group.ConsumeWithContext(ctx, []string{kafka.TopicReportDigest}, handler); err != nil {
					c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.ConsumeReportDigest: Consumer error: %v", err)
				}
			}
		}
	}()

	go func() {
		for err := range group.Errors() {
			c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.ConsumeReportDigest: Consumer group error: %v", err)
		}
	}()

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.ConsumeReportDigest: Started consuming topic: %s (group: %s)",
		kafka.TopicReportDigest, kafka.GroupIDReportDigest)

	return nil
}

// ConsumeInsightsPublished starts consuming analytics.insights.published messages
func (c *consumer) ConsumeInsightsPublished(ctx context.Context) error {
	group, err := c.createConsumerGroup(kafka.GroupIDInsightsPublished)
	if err != nil {
		return err
	}
	c.insightsPublishedGroup = group

	handler := &insightsPublishedHandler{consumer: c}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := group.ConsumeWithContext(ctx, []string{kafka.TopicInsightsPublished}, handler); err != nil {
					c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.ConsumeInsightsPublished: Consumer error: %v", err)
				}
			}
		}
	}()

	go func() {
		for err := range group.Errors() {
			c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.ConsumeInsightsPublished: Consumer group error: %v", err)
		}
	}()

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.ConsumeInsightsPublished: Started consuming topic: %s (group: %s)",
		kafka.TopicInsightsPublished, kafka.GroupIDInsightsPublished)

	return nil
}
