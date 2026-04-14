package consumer

import (
	"context"
	"fmt"
	"knowledge-srv/config"
	"knowledge-srv/internal/indexing"

	"github.com/smap-hcmut/shared-libs/go/kafka"
	"github.com/smap-hcmut/shared-libs/go/log"
)

// Consumer is the delivery interface for Kafka. Same idea as http.Handler: caller depends on interface, not concrete type.
type Consumer interface {
	ConsumeBatchCompleted(ctx context.Context) error
	ConsumeInsightsPublished(ctx context.Context) error
	ConsumeReportDigest(ctx context.Context) error
	Close() error
}

type Config struct {
	Logger      log.Logger
	KafkaConfig config.KafkaConfig
	UseCase     indexing.UseCase
}

// consumer implements Consumer (thin layer: receive msg → normalize → delegate to usecase).
type consumer struct {
	l                      log.Logger
	kafkaConfig            config.KafkaConfig
	uc                     indexing.UseCase
	batchCompletedGroup    kafka.IConsumer
	insightsPublishedGroup kafka.IConsumer
	reportDigestGroup      kafka.IConsumer
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
	var firstErr error

	if c.batchCompletedGroup != nil {
		if err := c.batchCompletedGroup.Close(); err != nil {
			c.l.Errorf(context.Background(), "indexing.delivery.kafka.consumer.Close: failed to close batch completed group: %v", err)
			firstErr = ErrConsumerGroupNotFound
		}
	}

	if c.insightsPublishedGroup != nil {
		if err := c.insightsPublishedGroup.Close(); err != nil {
			c.l.Errorf(context.Background(), "indexing.delivery.kafka.consumer.Close: failed to close insights published group: %v", err)
			if firstErr == nil {
				firstErr = ErrConsumerGroupNotFound
			}
		}
	}

	if c.reportDigestGroup != nil {
		if err := c.reportDigestGroup.Close(); err != nil {
			c.l.Errorf(context.Background(), "indexing.delivery.kafka.consumer.Close: failed to close report digest group: %v", err)
			if firstErr == nil {
				firstErr = ErrConsumerGroupNotFound
			}
		}
	}

	return firstErr
}

func (c *consumer) createConsumerGroup(groupID string) (kafka.IConsumer, error) {
	consumerConfig := kafka.ConsumerConfig{
		Brokers: c.kafkaConfig.Brokers,
		GroupID: groupID,
	}
	group, err := kafka.NewConsumer(consumerConfig)
	if err != nil {
		c.l.Errorf(context.Background(), "indexing.delivery.kafka.consumer.createConsumerGroup: failed to create consumer group %s: %v", groupID, err)
		return nil, ErrCreateConsumerGroupFailed
	}
	return group, nil
}
