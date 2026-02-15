package qdrant

import (
	"context"
	"fmt"
	"sync"
	"time"

	"knowledge-srv/config"
	"knowledge-srv/pkg/qdrant"
)

var (
	instance qdrant.IQdrant
	once     sync.Once
	mu       sync.RWMutex
	initErr  error
)

// Connect initializes and connects to Qdrant using singleton pattern.
func Connect(ctx context.Context, cfg config.QdrantConfig) (qdrant.IQdrant, error) {
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
		clientCfg := qdrant.Config{
			Host:    cfg.Host,
			Port:    cfg.Port,
			APIKey:  cfg.APIKey,
			UseTLS:  cfg.UseTLS,
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		}

		client, e := qdrant.NewQdrant(clientCfg)
		if e != nil {
			err = fmt.Errorf("failed to initialize Qdrant client: %w", e)
			initErr = err
			return
		}

		instance = client
	})

	return instance, err
}

// GetClient returns the singleton Qdrant client instance.
func GetClient() qdrant.IQdrant {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		panic("Qdrant client not initialized. Call Connect() first")
	}
	return instance
}

// HealthCheck checks if Qdrant connection is healthy
func HealthCheck(ctx context.Context) error {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		return fmt.Errorf("Qdrant client not initialized")
	}

	return instance.Ping(ctx)
}

// Disconnect closes the Qdrant connection
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
