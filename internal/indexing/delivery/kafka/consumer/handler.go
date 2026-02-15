package consumer

import (
	"github.com/IBM/sarama"
)

// ============================================
// Document Indexing Handler
// ============================================

type documentIndexingHandler struct {
	consumer *Consumer
}

func (h *documentIndexingHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *documentIndexingHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *documentIndexingHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.consumer.processDocumentIndexing(msg); err != nil {
			h.consumer.l.Errorf(nil, "Failed to process document indexing message: %v", err)
			// Continue processing other messages even if one fails
		}
		// Mark message as processed
		session.MarkMessage(msg, "")
	}
	return nil
}

// ============================================
// Knowledge Base Events Handler
// ============================================

type knowledgeBaseEventsHandler struct {
	consumer *Consumer
}

func (h *knowledgeBaseEventsHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *knowledgeBaseEventsHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *knowledgeBaseEventsHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.consumer.processKnowledgeBaseEvent(msg); err != nil {
			h.consumer.l.Errorf(nil, "Failed to process knowledge base event: %v", err)
			// Continue processing other messages even if one fails
		}
		// Mark message as processed
		session.MarkMessage(msg, "")
	}
	return nil
}
