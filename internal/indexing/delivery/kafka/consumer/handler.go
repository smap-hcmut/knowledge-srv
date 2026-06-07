package consumer

import (
	"context"

	"github.com/IBM/sarama"
)

type batchCompletedHandler struct {
	consumer *consumer
}

func (h *batchCompletedHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *batchCompletedHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *batchCompletedHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.consumer.handleBatchCompletedMessage(msg); err != nil {
			h.consumer.l.Errorf(context.Background(), "indexing.delivery.kafka.consumer.ConsumeBatchCompleted: Failed to process batch completed message: %v", err)
			if isPermanentError(err) {
				session.MarkMessage(msg, "")
			}
			continue
		}
		session.MarkMessage(msg, "")
	}
	return nil
}

type insightsPublishedHandler struct {
	consumer *consumer
}

func (h *insightsPublishedHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *insightsPublishedHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *insightsPublishedHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.consumer.handleInsightsPublishedMessage(msg); err != nil {
			h.consumer.l.Errorf(context.Background(), "indexing.delivery.kafka.consumer.ConsumeInsightsPublished: Failed to process message: %v", err)
			if isPermanentError(err) {
				session.MarkMessage(msg, "")
			}
			continue
		}
		session.MarkMessage(msg, "")
	}
	return nil
}

type reportDigestHandler struct {
	consumer *consumer
}

func (h *reportDigestHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *reportDigestHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *reportDigestHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.consumer.handleReportDigestMessage(msg); err != nil {
			h.consumer.l.Errorf(context.Background(), "indexing.delivery.kafka.consumer.ConsumeReportDigest: Failed to process message: %v", err)
			if isPermanentError(err) {
				session.MarkMessage(msg, "")
			}
			continue
		}
		session.MarkMessage(msg, "")
	}
	return nil
}

// isPermanentError returns true for errors that won't succeed on retry.
// Transient errors (DB timeout, Qdrant down) return false so the message
// stays uncommitted and the consumer group retries it.
func isPermanentError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// Permanent: parse errors, validation failures, unsupported types.
	permanent := []string{
		"json:", "unmarshal", "invalid payload", "unsupported",
		"unknown message type", "empty payload",
	}
	for _, p := range permanent {
		if len(msg) >= len(p) {
			// Simple substring match for the most common patterns.
			for i := 0; i <= len(msg)-len(p); i++ {
				if msg[i:i+len(p)] == p {
					return true
				}
			}
		}
	}
	return false
}
