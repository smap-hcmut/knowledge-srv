package minio

import (
	"io"
	"time"
)

// FileInfo represents metadata about a file stored in MinIO.
type FileInfo struct {
	ID           string            `json:"id"`
	BucketName   string            `json:"bucket_name"`
	ObjectName   string            `json:"object_name"`
	OriginalName string            `json:"original_name"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"content_type"`
	ETag         string            `json:"etag"`
	LastModified time.Time         `json:"last_modified"`
	Metadata     map[string]string `json:"metadata"`
	URL          string            `json:"url,omitempty"`

	// Compression metadata
	IsCompressed     bool    `json:"is_compressed"`
	CompressedSize   int64   `json:"compressed_size,omitempty"`
	UncompressedSize int64   `json:"uncompressed_size,omitempty"`
	CompressionRatio float64 `json:"compression_ratio,omitempty"`
}

// UploadRequest contains the parameters for uploading a file to MinIO.
type UploadRequest struct {
	BucketName   string            `json:"bucket_name"`
	ObjectName   string            `json:"object_name"`
	OriginalName string            `json:"original_name"`
	Reader       io.Reader         `json:"-"`
	Size         int64             `json:"size"`
	ContentType  string            `json:"content_type"`
	Metadata     map[string]string `json:"metadata"`

	// Compression options
	EnableCompression bool `json:"enable_compression"`
	CompressionLevel  int  `json:"compression_level,omitempty"` // 0=none, 1=fastest, 2=default, 3=best
}

// DownloadRequest contains the parameters for downloading a file from MinIO.
type DownloadRequest struct {
	BucketName  string     `json:"bucket_name"`
	ObjectName  string     `json:"object_name"`
	Range       *ByteRange `json:"range,omitempty"`
	Disposition string     `json:"disposition"` // "auto", "inline", "attachment"
}

// ByteRange represents a byte range for partial file downloads.
type ByteRange struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// ListRequest contains the parameters for listing files in a bucket.
type ListRequest struct {
	BucketName string `json:"bucket_name"`
	Prefix     string `json:"prefix"`
	Recursive  bool   `json:"recursive"`
	MaxKeys    int    `json:"max_keys"`
}

// ListResponse contains the result of listing files in a bucket.
type ListResponse struct {
	Files       []*FileInfo `json:"files"`
	IsTruncated bool        `json:"is_truncated"`
	NextMarker  string      `json:"next_marker,omitempty"`
	TotalCount  int         `json:"total_count"`
}

// PresignedURLRequest contains the parameters for generating a presigned URL.
type PresignedURLRequest struct {
	BucketName string            `json:"bucket_name"`
	ObjectName string            `json:"object_name"`
	Method     string            `json:"method"`
	Expiry     time.Duration     `json:"expiry"`
	Headers    map[string]string `json:"headers"`
}

// PresignedURLResponse contains the generated presigned URL and its metadata.
type PresignedURLResponse struct {
	URL       string            `json:"url"`
	ExpiresAt time.Time         `json:"expires_at"`
	Headers   map[string]string `json:"headers,omitempty"`
	Method    string            `json:"method"`
}

// DownloadHeaders contains HTTP headers for file download responses.
type DownloadHeaders struct {
	ContentType        string
	ContentDisposition string
	ContentLength      string
	LastModified       string
	ETag               string
	CacheControl       string
	AcceptRanges       string
	ContentRange       string
}

// BucketInfo contains information about a MinIO bucket.
type BucketInfo struct {
	Name         string    `json:"name"`
	CreationDate time.Time `json:"creation_date"`
	Region       string    `json:"region"`
}
