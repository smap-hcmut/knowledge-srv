package consumer

import (
	"context"
	kafkaDelivery "knowledge-srv/internal/indexing/delivery/kafka"
)

// ConsumeDocumentIndexing starts consuming document indexing messages
func (c *Consumer) ConsumeDocumentIndexing(ctx context.Context) error {
	// Create consumer group
	group, err := c.createConsumerGroup(kafkaDelivery.ConsumerGroupDocumentIndexing)
	if err != nil {
		return err
	}
	c.documentIndexingGroup = group

	// Create handler
	handler := &documentIndexingHandler{
		consumer: c,
	}

	// Start consuming in goroutine with context
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := group.ConsumeWithContext(ctx, []string{kafkaDelivery.TopicDocumentIndexing}, handler); err != nil {
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

	c.l.Infof(ctx, "Consuming %s", kafkaDelivery.TopicDocumentIndexing)

	return nil
}

// ConsumeKnowledgeBaseEvents starts consuming knowledge base events
func (c *Consumer) ConsumeKnowledgeBaseEvents(ctx context.Context) error {
	// Create consumer group
	group, err := c.createConsumerGroup(kafkaDelivery.ConsumerGroupKnowledgeBaseEvents)
	if err != nil {
		return err
	}
	c.knowledgeBaseEventsGroup = group

	// Create handler
	handler := &knowledgeBaseEventsHandler{
		consumer: c,
	}

	// Start consuming in goroutine with context
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := group.ConsumeWithContext(ctx, []string{kafkaDelivery.TopicKnowledgeBaseEvents}, handler); err != nil {
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

	c.l.Infof(ctx, "Consuming %s", kafkaDelivery.TopicKnowledgeBaseEvents)

	return nil
}
