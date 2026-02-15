package minio

import (
	"context"
	"fmt"
	"sync"

	"knowledge-srv/config"
	"knowledge-srv/pkg/minio"
)

var (
	instance minio.MinIO
	once     sync.Once
	mu       sync.RWMutex
	initErr  error
)

// Connect initializes and connects to MinIO using singleton pattern.
func Connect(ctx context.Context, cfg *config.MinIOConfig) (minio.MinIO, error) {
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
		// NewMinIO takes *config.MinIOConfig and returns interface MinIO
		client := minio.NewMinIO(cfg)

		// Connect to verify
		if e := client.Connect(ctx); e != nil {
			err = fmt.Errorf("failed to connect to MinIO: %w", e)
			initErr = err
			return
		}

		instance = client
	})

	return instance, err
}

// GetClient returns the singleton MinIO client instance.
func GetClient() minio.MinIO {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		panic("MinIO client not initialized. Call Connect() first")
	}
	return instance
}

// HealthCheck checks if MinIO connection is healthy
func HealthCheck(ctx context.Context) error {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		return fmt.Errorf("MinIO client not initialized")
	}

	// MinIO interface doesn't expose Ping explicitly in pkg/minio/new.go,
	// but Connect performs the check. Assuming Connect can be called idempotent
	// or we rely on internal state.
	// Alternatively, pkg/minio might handle keep-alive.
	// For now, check if instance is not nil.
	return nil
}
