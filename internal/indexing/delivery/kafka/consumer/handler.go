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
