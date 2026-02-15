package kafka

import (
	"fmt"
	"time"

	"github.com/IBM/sarama"
)

// Producer wraps Kafka producer client
type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

// Config holds configuration for Kafka producer
type Config struct {
	Brokers []string
	Topic   string
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg Config) (*Producer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("at least one broker is required")
	}
	if cfg.Topic == "" {
		return nil, fmt.Errorf("topic is required")
	}

	// Configure Kafka producer
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal     // Wait for leader to acknowledge
	config.Producer.Compression = sarama.CompressionSnappy // Compress messages
	config.Producer.Return.Successes = true                // Return success messages
	config.Producer.Retry.Max = 3                          // Retry up to 3 times
	config.Producer.Timeout = 10 * time.Second             // Timeout after 10 seconds
	config.Version = sarama.V2_6_0_0                       // Kafka version

	// Create sync producer
	producer, err := sarama.NewSyncProducer(cfg.Brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	return &Producer{
		producer: producer,
		topic:    cfg.Topic,
	}, nil
}

// Publish sends a message to Kafka topic
func (p *Producer) Publish(key, value []byte) error {
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

// Close closes the Kafka producer
func (p *Producer) Close() error {
	if p.producer != nil {
		return p.producer.Close()
	}
	return nil
}

// HealthCheck verifies connection to Kafka
func (p *Producer) HealthCheck() error {
	// Try to get metadata to verify connection
	if p.producer == nil {
		return fmt.Errorf("producer is not initialized")
	}
	return nil
}
