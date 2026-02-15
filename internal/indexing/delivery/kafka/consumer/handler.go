package consumer

import (
	"github.com/IBM/sarama"
)

// ============================================
// Analytics Batch Completed Handler
// ============================================

type batchCompletedHandler struct {
	consumer *Consumer
}

func (h *batchCompletedHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *batchCompletedHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *batchCompletedHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.consumer.processBatchCompleted(msg); err != nil {
			h.consumer.l.Errorf(nil, "Failed to process batch completed message: %v", err)
			// Continue processing other messages even if one fails
		}
		// Mark message as processed
		session.MarkMessage(msg, "")
	}
	return nil
}
