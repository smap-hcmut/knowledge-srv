package redis

import (
	"context"
	"fmt"
	"sync"

	"knowledge-srv/config"

	"github.com/smap-hcmut/shared-libs/go/redis"
)

var (
	instance redis.IRedis
	once     sync.Once
	mu       sync.RWMutex
	initErr  error
)

// Connect initializes and connects to Redis using singleton pattern.
func Connect(ctx context.Context, cfg config.RedisConfig) (redis.IRedis, error) {
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
		clientCfg := redis.RedisConfig{
			Host:     cfg.Host,
			Port:     cfg.Port,
			Password: cfg.Password,
			DB:       cfg.DB,
		}

		client, e := redis.New(clientCfg)
		if e != nil {
			err = fmt.Errorf("failed to initialize Redis client: %w", e)
			initErr = err
			return
		}

		if e := client.Ping(ctx); e != nil {
			_ = client.Close()
			err = fmt.Errorf("failed to ping Redis: %w", e)
			initErr = err
			return
		}

		instance = client
	})

	return instance, err
}

// Disconnect closes the Redis connection
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
