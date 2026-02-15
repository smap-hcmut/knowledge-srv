package consumer

import (
	"fmt"

	"knowledge-srv/config"
	"knowledge-srv/internal/indexing"
	pkgKafka "knowledge-srv/pkg/kafka"
	"knowledge-srv/pkg/log"
)

// Config holds the configuration for indexing consumer
type Config struct {
	Logger      log.Logger
	KafkaConfig config.KafkaConfig
	UseCase     indexing.UseCase
}

// Consumer manages Kafka consumer groups for indexing domain
type Consumer struct {
	l           log.Logger
	kafkaConfig config.KafkaConfig
	uc          indexing.UseCase

	// Consumer group for analytics batch completed
	batchCompletedGroup pkgKafka.IConsumer
}

// New creates a new indexing consumer
func New(cfg Config) (*Consumer, error) {
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if cfg.UseCase == nil {
		return nil, fmt.Errorf("usecase is required")
	}
	if len(cfg.KafkaConfig.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}

	return &Consumer{
		l:           cfg.Logger,
		kafkaConfig: cfg.KafkaConfig,
		uc:          cfg.UseCase,
	}, nil
}

// Close closes all consumer groups
func (c *Consumer) Close() error {
	if c.batchCompletedGroup != nil {
		if err := c.batchCompletedGroup.Close(); err != nil {
			return fmt.Errorf("failed to close batch completed group: %w", err)
		}
	}

	return nil
}

// createConsumerGroup creates a new Kafka consumer group
func (c *Consumer) createConsumerGroup(groupID string) (pkgKafka.IConsumer, error) {
	consumerConfig := pkgKafka.ConsumerConfig{
		Brokers: c.kafkaConfig.Brokers,
		GroupID: groupID,
	}

	group, err := pkgKafka.NewConsumer(consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group %s: %w", groupID, err)
	}

	return group, nil
}
