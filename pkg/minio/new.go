package minio

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"knowledge-srv/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	maxIdleConns        = 100
	maxIdleConnsPerHost = 100
	idleConnTimeout     = 90 * time.Second
	disableCompression  = true
	disableKeepAlives   = false
)

// MinIO defines the interface for MinIO storage operations
type MinIO interface {
	// Connection Management
	// Connect establishes a connection to MinIO and verifies it's working
	Connect(ctx context.Context) error

	// ConnectWithRetry establishes a connection with retry logic and exponential backoff
	ConnectWithRetry(ctx context.Context, maxRetries int) error

	// HealthCheck verifies the connection is still healthy
	HealthCheck(ctx context.Context) error

	// Close closes the connection and cleans up resources
	Close() error

	// Bucket Operations
	// CreateBucket creates a new bucket with the specified name
	CreateBucket(ctx context.Context, bucketName string) error

	// DeleteBucket removes a bucket and all its contents
	DeleteBucket(ctx context.Context, bucketName string) error

	// BucketExists checks if a bucket exists
	BucketExists(ctx context.Context, bucketName string) (bool, error)

	// ListBuckets returns all available buckets
	ListBuckets(ctx context.Context) ([]*BucketInfo, error)

	// File Upload Operations
	// UploadFile uploads a file to MinIO storage
	UploadFile(ctx context.Context, req *UploadRequest) (*FileInfo, error)

	// GetPresignedUploadURL generates a presigned URL for direct upload
	GetPresignedUploadURL(ctx context.Context, req *PresignedURLRequest) (*PresignedURLResponse, error)

	// File Download Operations
	// DownloadFile downloads a file from MinIO storage
	DownloadFile(ctx context.Context, req *DownloadRequest) (io.ReadCloser, *DownloadHeaders, error)

	// StreamFile streams a file for web viewing (optimized for streaming)
	StreamFile(ctx context.Context, req *DownloadRequest) (io.ReadCloser, *DownloadHeaders, error)

	// GetPresignedDownloadURL generates a presigned URL for direct download
	GetPresignedDownloadURL(ctx context.Context, req *PresignedURLRequest) (*PresignedURLResponse, error)

	// File Management Operations
	// GetFileInfo retrieves metadata about a file
	GetFileInfo(ctx context.Context, bucketName, objectName string) (*FileInfo, error)

	// DeleteFile removes a file from storage
	DeleteFile(ctx context.Context, bucketName, objectName string) error

	// CopyFile copies a file from one location to another
	CopyFile(ctx context.Context, srcBucket, srcObject, destBucket, destObject string) error

	// MoveFile moves a file from one location to another (copy + delete)
	MoveFile(ctx context.Context, srcBucket, srcObject, destBucket, destObject string) error

	// FileExists checks if a file exists
	FileExists(ctx context.Context, bucketName, objectName string) (bool, error)

	// File Listing Operations
	// ListFiles lists files in a bucket with optional filtering
	ListFiles(ctx context.Context, req *ListRequest) (*ListResponse, error)

	// Metadata Operations
	// UpdateMetadata updates the metadata of a file
	UpdateMetadata(ctx context.Context, bucketName, objectName string, metadata map[string]string) error

	// GetMetadata retrieves the metadata of a file
	GetMetadata(ctx context.Context, bucketName, objectName string) (map[string]string, error)

	// Async Upload Operations
	// UploadAsync queues an upload task and returns task ID
	UploadAsync(ctx context.Context, req *UploadRequest) (taskID string, err error)

	// GetUploadStatus retrieves upload status by task ID
	GetUploadStatus(taskID string) (*UploadProgress, error)

	// WaitForUpload waits for upload completion with timeout
	WaitForUpload(taskID string, timeout time.Duration) (*AsyncUploadResult, error)

	// CancelUpload cancels an upload task
	CancelUpload(taskID string) error
}

// implMinIO is the implementation of the MinIO interface.
type implMinIO struct {
	minioClient    *minio.Client
	config         *config.MinIOConfig
	mu             sync.RWMutex
	connected      bool
	asyncUploadMgr *AsyncUploadManager
}

// NewMinIO creates a new MinIO client with the provided configuration.
// NOTE: This function creates a new instance each time. For singleton pattern,
// use Connect() or ConnectWithRetry() instead.
// Deprecated: Use Connect() or ConnectWithRetry() for singleton pattern.
func NewMinIO(config *config.MinIOConfig) (MinIO, error) {
	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	// Create optimized HTTP transport
	transport := &http.Transport{
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: maxIdleConnsPerHost,
		IdleConnTimeout:     idleConnTimeout,
		DisableCompression:  disableCompression,
		DisableKeepAlives:   disableKeepAlives,
	}

	// Create MinIO client
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure:    config.UseSSL,
		Region:    config.Region,
		Transport: transport,
	})
	if err != nil {
		return nil, err
	}

	impl := &implMinIO{
		minioClient: client,
		config:      config,
		connected:   false,
	}

	// Initialize async upload manager
	workerPoolSize := 4 // default
	queueSize := 100    // default
	if config.AsyncUploadWorkers > 0 {
		workerPoolSize = config.AsyncUploadWorkers
	}
	if config.AsyncUploadQueueSize > 0 {
		queueSize = config.AsyncUploadQueueSize
	}

	impl.asyncUploadMgr = NewAsyncUploadManager(impl, workerPoolSize, queueSize)
	impl.asyncUploadMgr.Start()

	return impl, nil
}

// NewMinIOWithRetry creates a new MinIO client and connects with retry logic.
// NOTE: This function creates a new instance each time. For singleton pattern,
// use ConnectWithRetry() instead.
// Deprecated: Use ConnectWithRetry() for singleton pattern.
func NewMinIOWithRetry(config *config.MinIOConfig, maxRetries int) (MinIO, error) {
	client, err := NewMinIO(config)
	if err != nil {
		return nil, err
	}

	// Connect with retry
	if err := client.ConnectWithRetry(context.Background(), maxRetries); err != nil {
		return nil, err
	}

	return client, nil
}
