package qdrant

import (
	"context"
	"fmt"
	"time"

	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps Qdrant client with common operations
type Client struct {
	conn              *grpc.ClientConn
	pointsClient      pb.PointsClient
	collectionsClient pb.CollectionsClient
	defaultTimeout    time.Duration
}

// New creates a new Qdrant client
func New(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Set default timeout if not specified
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	// Create gRPC connection
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	var opts []grpc.DialOption
	if cfg.UseTLS {
		// Add TLS credentials if needed
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	client := &Client{
		conn:              conn,
		pointsClient:      pb.NewPointsClient(conn),
		collectionsClient: pb.NewCollectionsClient(conn),
		defaultTimeout:    cfg.Timeout,
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping Qdrant: %w", err)
	}

	return client, nil
}

// Close closes the Qdrant connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Ping checks if Qdrant is reachable
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.collectionsClient.List(ctx, &pb.ListCollectionsRequest{})
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	return nil
}

// GetPointsClient returns the underlying points client for advanced operations
func (c *Client) GetPointsClient() pb.PointsClient {
	return c.pointsClient
}

// GetCollectionsClient returns the underlying collections client for advanced operations
func (c *Client) GetCollectionsClient() pb.CollectionsClient {
	return c.collectionsClient
}

// GetDefaultTimeout returns the default timeout for operations
func (c *Client) GetDefaultTimeout() time.Duration {
	return c.defaultTimeout
}
