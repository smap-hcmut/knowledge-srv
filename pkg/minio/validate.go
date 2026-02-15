package minio

import (
	"strings"
	"time"

	"smap-project/config"
)

// validateConfig validates the MinIO configuration
func validateConfig(cfg *config.MinIOConfig) error {
	if cfg.Endpoint == "" {
		return NewInvalidInputError("endpoint is required")
	}
	if cfg.AccessKey == "" {
		return NewInvalidInputError("access key is required")
	}
	if cfg.SecretKey == "" {
		return NewInvalidInputError("secret key is required")
	}
	if cfg.Region == "" {
		return NewInvalidInputError("region is required")
	}
	if cfg.Bucket == "" {
		return NewInvalidInputError("bucket is required")
	}

	if !strings.Contains(cfg.Endpoint, ":") {
		cfg.Endpoint = cfg.Endpoint + ":9000"
	}

	return nil
}

// validateUploadRequest validates upload request parameters
func validateUploadRequest(req *UploadRequest) error {
	if req.BucketName == "" {
		return NewInvalidInputError("bucket name is required")
	}
	if req.ObjectName == "" {
		return NewInvalidInputError("object name is required")
	}
	if req.Reader == nil {
		return NewInvalidInputError("reader is required")
	}
	if req.Size <= 0 {
		return NewInvalidInputError("size must be positive")
	}
	if req.ContentType == "" {
		return NewInvalidInputError("content type is required")
	}

	// Validate object name format
	if strings.HasPrefix(req.ObjectName, "/") {
		return NewInvalidInputError("object name cannot start with '/'")
	}
	if strings.HasSuffix(req.ObjectName, "/") {
		return NewInvalidInputError("object name cannot end with '/'")
	}

	// Validate size limits (optional)
	if req.Size > 5*1024*1024*1024 { // 5GB
		return NewInvalidInputError("file size cannot exceed 5GB")
	}

	return nil
}

// validateDownloadRequest validates download request parameters
func validateDownloadRequest(req *DownloadRequest) error {
	if req.BucketName == "" {
		return NewInvalidInputError("bucket name is required")
	}
	if req.ObjectName == "" {
		return NewInvalidInputError("object name is required")
	}

	// Validate disposition
	if req.Disposition != "" && req.Disposition != "auto" && req.Disposition != "inline" && req.Disposition != "attachment" {
		return NewInvalidInputError("disposition must be 'auto', 'inline', or 'attachment'")
	}

	// Validate range if provided
	if req.Range != nil {
		if req.Range.Start < 0 {
			return NewInvalidInputError("range start must be non-negative")
		}
		if req.Range.End < req.Range.Start {
			return NewInvalidInputError("range end must be greater than or equal to start")
		}
	}

	return nil
}

// validateListRequest validates list request parameters
func validateListRequest(req *ListRequest) error {
	if req.BucketName == "" {
		return NewInvalidInputError("bucket name is required")
	}

	// Set default max keys if not specified
	if req.MaxKeys <= 0 {
		req.MaxKeys = 1000
	}
	if req.MaxKeys > 1000 {
		return NewInvalidInputError("max keys cannot exceed 1000")
	}

	return nil
}

// validatePresignedURLRequest validates presigned URL request parameters
func validatePresignedURLRequest(req *PresignedURLRequest) error {
	if req.BucketName == "" {
		return NewInvalidInputError("bucket name is required")
	}
	if req.ObjectName == "" {
		return NewInvalidInputError("object name is required")
	}
	if req.Method == "" {
		return NewInvalidInputError("method is required")
	}
	if req.Method != "GET" && req.Method != "PUT" {
		return NewInvalidInputError("method must be 'GET' or 'PUT'")
	}
	if req.Expiry <= 0 {
		return NewInvalidInputError("expiry must be positive")
	}
	if req.Expiry > 7*24*time.Hour { // 7 days
		return NewInvalidInputError("expiry cannot exceed 7 days")
	}

	return nil
}

// validateBucketName validates bucket name format
func validateBucketName(bucketName string) error {
	if bucketName == "" {
		return NewInvalidInputError("bucket name is required")
	}

	// MinIO bucket naming rules
	if len(bucketName) < 3 {
		return NewInvalidInputError("bucket name must be at least 3 characters")
	}
	if len(bucketName) > 63 {
		return NewInvalidInputError("bucket name cannot exceed 63 characters")
	}

	// Check for valid characters
	for _, char := range bucketName {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-') {
			return NewInvalidInputError("bucket name can only contain lowercase letters, numbers, and hyphens")
		}
	}

	// Check for consecutive hyphens
	if strings.Contains(bucketName, "--") {
		return NewInvalidInputError("bucket name cannot contain consecutive hyphens")
	}

	// Check for start/end with hyphen
	if strings.HasPrefix(bucketName, "-") || strings.HasSuffix(bucketName, "-") {
		return NewInvalidInputError("bucket name cannot start or end with hyphen")
	}

	return nil
}

// validateObjectName validates object name format
func validateObjectName(objectName string) error {
	if objectName == "" {
		return NewInvalidInputError("object name is required")
	}

	// Check for valid characters (basic validation)
	if strings.Contains(objectName, "\\") {
		return NewInvalidInputError("object name cannot contain backslashes")
	}

	return nil
}
