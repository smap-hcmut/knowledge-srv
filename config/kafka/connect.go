package kafka

import (
	"fmt"
	"sync"

	"knowledge-srv/config"
	"knowledge-srv/pkg/kafka"
)

var (
	instance *kafka.Producer
	once     sync.Once
	mu       sync.RWMutex
	initErr  error
)

// Connect initializes and connects to Kafka using singleton pattern.
func Connect(cfg config.KafkaConfig) (*kafka.Producer, error) {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		return instance, nil
	}

	if initErr != nil {
		once = sync.Once{}
		initErr = nil
	}

	var err error
	once.Do(func() {
		clientCfg := kafka.Config{
			Brokers: cfg.Brokers,
			Topic:   cfg.Topic,
		}

		client, e := kafka.NewProducer(clientCfg)
		if e != nil {
			err = fmt.Errorf("failed to initialize Kafka producer: %w", e)
			initErr = err
			return
		}

		instance = client
	})

	return instance, err
}

// GetClient returns the singleton Kafka producer instance.
func GetClient() *kafka.Producer {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		panic("Kafka producer not initialized. Call Connect() first")
	}
	return instance
}

// HealthCheck checks if Kafka is initialized
func HealthCheck() error {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		return fmt.Errorf("Kafka producer not initialized")
	}
	return instance.HealthCheck()
}

// Disconnect closes the Kafka producer
func Disconnect() error {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		if err := instance.Close(); err != nil {
			return err
		}
		instance = nil
		once = sync.Once{}
		initErr = nil
	}
	return nil
}
