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
			continue
		}
		session.MarkMessage(msg, "")
	}
	return nil
}
