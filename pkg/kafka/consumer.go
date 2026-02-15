package kafka

import (
	"fmt"

	"github.com/IBM/sarama"
)

// ConsumerConfig holds configuration for Kafka consumer
type ConsumerConfig struct {
	Brokers []string
	GroupID string
}

// NewConsumerGroup creates a new Kafka consumer group
func NewConsumerGroup(cfg ConsumerConfig) (sarama.ConsumerGroup, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("at least one broker is required")
	}
	if cfg.GroupID == "" {
		return nil, fmt.Errorf("group ID is required")
	}

	// Configure Kafka consumer
	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Return.Errors = true

	// Create consumer group
	consumer, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer group: %w", err)
	}

	return consumer, nil
}
