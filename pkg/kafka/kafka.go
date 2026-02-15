package kafka

import (
	"fmt"

	"github.com/IBM/sarama"
)

func validateProducerConfig(cfg Config) error {
	if len(cfg.Brokers) == 0 {
		return fmt.Errorf("kafka: at least one broker is required")
	}
	if cfg.Topic == "" {
		return fmt.Errorf("kafka: topic is required")
	}
	return nil
}

func validateConsumerConfig(cfg ConsumerConfig) error {
	if len(cfg.Brokers) == 0 {
		return fmt.Errorf("kafka: at least one broker is required")
	}
	if cfg.GroupID == "" {
		return fmt.Errorf("kafka: group ID is required")
	}
	return nil
}

func newProducerImpl(cfg Config) (*producerImpl, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Compression = sarama.CompressionSnappy
	config.Producer.Return.Successes = true
	config.Producer.Retry.Max = ProducerRetryMax
	config.Producer.Timeout = ProducerTimeout
	config.Version = KafkaVersion

	producer, err := sarama.NewSyncProducer(cfg.Brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}
	return &producerImpl{producer: producer, topic: cfg.Topic}, nil
}

// Publish sends a message to the configured topic.
func (p *producerImpl) Publish(key, value []byte) error {
	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	}
	_, _, err := p.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to publish message to Kafka: %w", err)
	}
	return nil
}

// Close closes the producer.
func (p *producerImpl) Close() error {
	if p.producer != nil {
		return p.producer.Close()
	}
	return nil
}

// HealthCheck verifies the producer is initialized.
func (p *producerImpl) HealthCheck() error {
	if p.producer == nil {
		return fmt.Errorf("producer is not initialized")
	}
	return nil
}

// NewConsumerGroup creates a new Kafka consumer group.
func NewConsumerGroup(cfg ConsumerConfig) (sarama.ConsumerGroup, error) {
	if err := validateConsumerConfig(cfg); err != nil {
		return nil, err
	}
	config := sarama.NewConfig()
	config.Version = KafkaVersion
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Return.Errors = true
	consumer, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer group: %w", err)
	}
	return consumer, nil
}
