package consumer

import (
	"context"
	"fmt"

	"knowledge-srv/config"
	"knowledge-srv/internal/indexing"
	pkgKafka "knowledge-srv/pkg/kafka"
	"knowledge-srv/pkg/log"
)

// Consumer is the delivery interface for Kafka. Same idea as http.Handler: caller depends on interface, not concrete type.
type Consumer interface {
	ConsumeBatchCompleted(ctx context.Context) error
	Close() error
}

type Config struct {
	Logger      log.Logger
	KafkaConfig config.KafkaConfig
	UseCase     indexing.UseCase
}

// consumer implements Consumer (thin layer: receive msg → normalize → delegate to usecase).
type consumer struct {
	l           log.Logger
	kafkaConfig config.KafkaConfig
	uc          indexing.UseCase

	batchCompletedGroup pkgKafka.IConsumer
}

func New(cfg Config) (Consumer, error) {
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if cfg.UseCase == nil {
		return nil, fmt.Errorf("usecase is required")
	}
	if len(cfg.KafkaConfig.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}

	return &consumer{
		l:           cfg.Logger,
		kafkaConfig: cfg.KafkaConfig,
		uc:          cfg.UseCase,
	}, nil
}

func (c *consumer) Close() error {
	if c.batchCompletedGroup != nil {
		if err := c.batchCompletedGroup.Close(); err != nil {
			c.l.Errorf(context.Background(), "indexing.delivery.kafka.consumer.Close: failed to close batch completed group: %v", err)
			return ErrConsumerGroupNotFound
		}
	}
	return nil
}

func (c *consumer) createConsumerGroup(groupID string) (pkgKafka.IConsumer, error) {
	consumerConfig := pkgKafka.ConsumerConfig{
		Brokers: c.kafkaConfig.Brokers,
		GroupID: groupID,
	}
	group, err := pkgKafka.NewConsumer(consumerConfig)
	if err != nil {
		c.l.Errorf(context.Background(), "indexing.delivery.kafka.consumer.createConsumerGroup: failed to create consumer group %s: %v", groupID, err)
		return nil, ErrCreateConsumerGroupFailed
	}
	return group, nil
}
