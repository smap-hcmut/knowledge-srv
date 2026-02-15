package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"knowledge-srv/internal/indexing"
	kafkaDelivery "knowledge-srv/internal/indexing/delivery/kafka"
	"knowledge-srv/internal/model"

	"github.com/IBM/sarama"
)

// processDocumentIndexing processes a document indexing message
func (c *Consumer) processDocumentIndexing(msg *sarama.ConsumerMessage) error {
	ctx := context.Background()

	c.l.Infof(ctx, "Processing document indexing message from partition %d, offset %d",
		msg.Partition, msg.Offset)

	// 1. Unmarshal message
	var message kafkaDelivery.DocumentIndexingMessage
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		c.l.Warnf(ctx, "Invalid message format (skipping): %v", err)
		return nil // Skip invalid messages
	}

	// 2. Validate message
	if message.DocumentID == "" || message.KnowledgeBaseID == "" {
		c.l.Warnf(ctx, "Invalid message: missing required fields (skipping)")
		return nil
	}

	// 3. Set default values
	if message.ChunkSize == 0 {
		message.ChunkSize = 1000 // Default chunk size
	}
	if message.ChunkOverlap == 0 {
		message.ChunkOverlap = 200 // Default overlap
	}

	// 4. Convert to UseCase input
	input := indexing.ProcessDocumentInput{
		DocumentID:      message.DocumentID,
		KnowledgeBaseID: message.KnowledgeBaseID,
		FilePath:        message.FilePath,
		FileType:        message.FileType,
		ChunkSize:       message.ChunkSize,
		ChunkOverlap:    message.ChunkOverlap,
	}

	// 5. Create scope (system scope for background processing)
	scope := model.Scope{
		UserID: "system",
		Role:   "system",
	}

	// 6. Call UseCase
	if err := c.uc.ProcessDocumentIndexing(ctx, scope, input); err != nil {
		c.l.Errorf(ctx, "Failed to process document indexing: %v", err)
		return fmt.Errorf("usecase error: %w", err)
	}

	c.l.Infof(ctx, "Successfully processed document indexing for document %s", message.DocumentID)
	return nil
}

// processKnowledgeBaseEvent processes a knowledge base event message
func (c *Consumer) processKnowledgeBaseEvent(msg *sarama.ConsumerMessage) error {
	ctx := context.Background()

	c.l.Infof(ctx, "Processing knowledge base event from partition %d, offset %d",
		msg.Partition, msg.Offset)

	// 1. Unmarshal message
	var message kafkaDelivery.KnowledgeBaseEventMessage
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		c.l.Warnf(ctx, "Invalid message format (skipping): %v", err)
		return nil // Skip invalid messages
	}

	// 2. Route based on event type
	switch message.EventType {
	case kafkaDelivery.EventTypeKnowledgeBaseCreated:
		return c.handleKnowledgeBaseCreated(ctx, message)
	case kafkaDelivery.EventTypeKnowledgeBaseUpdated:
		return c.handleKnowledgeBaseUpdated(ctx, message)
	case kafkaDelivery.EventTypeKnowledgeBaseDeleted:
		return c.handleKnowledgeBaseDeleted(ctx, message)
	default:
		c.l.Warnf(ctx, "Unknown event type: %s (skipping)", message.EventType)
		return nil
	}
}

// handleKnowledgeBaseCreated handles knowledge base created event
func (c *Consumer) handleKnowledgeBaseCreated(ctx context.Context, msg kafkaDelivery.KnowledgeBaseEventMessage) error {
	event := indexing.KnowledgeBaseCreatedEvent{
		KnowledgeBaseID: msg.KnowledgeBaseID,
		ProjectID:       msg.ProjectID,
		Name:            msg.Name,
		Description:     msg.Description,
		CreatedBy:       msg.UserID,
		CreatedAt:       msg.Timestamp,
	}

	if err := c.uc.HandleKnowledgeBaseCreated(ctx, event); err != nil {
		c.l.Errorf(ctx, "Failed to handle knowledge base created: %v", err)
		return fmt.Errorf("usecase error: %w", err)
	}

	c.l.Infof(ctx, "Successfully handled knowledge base created: %s", msg.KnowledgeBaseID)
	return nil
}

// handleKnowledgeBaseUpdated handles knowledge base updated event
func (c *Consumer) handleKnowledgeBaseUpdated(ctx context.Context, msg kafkaDelivery.KnowledgeBaseEventMessage) error {
	event := indexing.KnowledgeBaseUpdatedEvent{
		KnowledgeBaseID: msg.KnowledgeBaseID,
		Name:            msg.Name,
		Description:     msg.Description,
		UpdatedBy:       msg.UserID,
		UpdatedAt:       msg.Timestamp,
	}

	if err := c.uc.HandleKnowledgeBaseUpdated(ctx, event); err != nil {
		c.l.Errorf(ctx, "Failed to handle knowledge base updated: %v", err)
		return fmt.Errorf("usecase error: %w", err)
	}

	c.l.Infof(ctx, "Successfully handled knowledge base updated: %s", msg.KnowledgeBaseID)
	return nil
}

// handleKnowledgeBaseDeleted handles knowledge base deleted event
func (c *Consumer) handleKnowledgeBaseDeleted(ctx context.Context, msg kafkaDelivery.KnowledgeBaseEventMessage) error {
	event := indexing.KnowledgeBaseDeletedEvent{
		KnowledgeBaseID: msg.KnowledgeBaseID,
		DeletedBy:       msg.UserID,
		DeletedAt:       msg.Timestamp,
	}

	if err := c.uc.HandleKnowledgeBaseDeleted(ctx, event); err != nil {
		c.l.Errorf(ctx, "Failed to handle knowledge base deleted: %v", err)
		return fmt.Errorf("usecase error: %w", err)
	}

	c.l.Infof(ctx, "Successfully handled knowledge base deleted: %s", msg.KnowledgeBaseID)
	return nil
}
