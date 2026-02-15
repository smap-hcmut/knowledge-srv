package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps Redis client with common operations
type Client struct {
	client *redis.Client
}

// Config holds Redis configuration
type Config struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// New creates a new Redis client
func New(cfg Config) (*Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{client: client}, nil
}

// Set stores a key-value pair with TTL
func (c *Client) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Get retrieves a value by key
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// Delete removes a key
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// TTL returns the remaining TTL of a key
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.client.Close()
}

// Ping checks if Redis is reachable
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// GetClient returns the underlying redis.Client for advanced operations
func (c *Client) GetClient() *redis.Client {
	return c.client
}
