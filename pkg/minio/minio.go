package minio

import (
	"context"
	"fmt"
	"io"
	"time"

	"smap-project/pkg/compressor"

	"github.com/minio/minio-go/v7"
)

// Connect establishes a connection to MinIO and verifies it's working by listing buckets.
func (m *implMinIO) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.minioClient.ListBuckets(ctx)
	if err != nil {
		m.connected = false
		return handleMinIOError(err, "connect")
	}

	m.connected = true
	return nil
}

// ConnectWithRetry establishes a connection with retry logic and exponential backoff.
func (m *implMinIO) ConnectWithRetry(ctx context.Context, maxRetries int) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := m.Connect(ctx); err == nil {
			return nil
		} else {
			lastErr = err
			// Exponential backoff
			backoff := time.Duration(1<<uint(i)) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}
	}
	return fmt.Errorf("failed to connect after %d retries: %w", maxRetries, lastErr)
}

// HealthCheck verifies the connection is still healthy by listing buckets.
func (m *implMinIO) HealthCheck(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.connected {
		return NewConnectionError(fmt.Errorf("not connected"))
	}

	_, err := m.minioClient.ListBuckets(ctx)
	if err != nil {
		return handleMinIOError(err, "health_check")
	}

	return nil
}

// Close closes the MinIO connection and marks it as disconnected.
// The MinIO client automatically manages the connection pool, so no explicit shutdown is required.
func (m *implMinIO) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.connected = false
	return nil
}

// CreateBucket creates a new bucket with the specified name.
// Returns an error if the bucket already exists or if creation fails.
func (m *implMinIO) CreateBucket(ctx context.Context, bucketName string) error {
	if err := validateBucketName(bucketName); err != nil {
		return err
	}

	exists, err := m.minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return handleMinIOError(err, "check_bucket_exists")
	}
	if exists {
		return NewInvalidInputError(fmt.Sprintf("bucket already exists: %s", bucketName))
	}

	err = m.minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: m.config.Region})
	if err != nil {
		return handleMinIOError(err, "create_bucket")
	}

	return nil
}

// DeleteBucket removes a bucket and all its contents.
func (m *implMinIO) DeleteBucket(ctx context.Context, bucketName string) error {
	if err := validateBucketName(bucketName); err != nil {
		return err
	}

	err := m.minioClient.RemoveBucket(ctx, bucketName)
	if err != nil {
		return handleMinIOError(err, "delete_bucket")
	}
	return nil
}

// BucketExists checks if a bucket exists.
func (m *implMinIO) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	if err := validateBucketName(bucketName); err != nil {
		return false, err
	}

	exists, err := m.minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return false, handleMinIOError(err, "check_bucket_exists")
	}
	return exists, nil
}

// ListBuckets returns all available buckets.
func (m *implMinIO) ListBuckets(ctx context.Context) ([]*BucketInfo, error) {
	buckets, err := m.minioClient.ListBuckets(ctx)
	if err != nil {
		return nil, handleMinIOError(err, "list_buckets")
	}

	var result []*BucketInfo
	for _, bucket := range buckets {
		result = append(result, &BucketInfo{
			Name:         bucket.Name,
			CreationDate: bucket.CreationDate,
			Region:       m.config.Region,
		})
	}

	return result, nil
}

// UploadFile uploads a file to MinIO storage.
func (m *implMinIO) UploadFile(ctx context.Context, req *UploadRequest) (*FileInfo, error) {
	if err := validateUploadRequest(req); err != nil {
		return nil, err
	}

	var reader io.Reader = req.Reader
	var actualSize int64 = req.Size
	var isCompressed bool
	var uncompressedSize int64

	// Apply compression if enabled
	if req.EnableCompression {
		// Map compression level
		var compLevel compressor.CompressionLevel
		switch req.CompressionLevel {
		case 1:
			compLevel = compressor.CompressionFastest
		case 3:
			compLevel = compressor.CompressionBest
		default:
			compLevel = compressor.CompressionDefault
		}

		// Compress using streaming compression
		compressedReader, err := compressor.CompressStream(req.Reader, compLevel)
		if err != nil {
			return nil, fmt.Errorf("compression failed: %w", err)
		}
		defer compressedReader.Close()

		reader = compressedReader
		uncompressedSize = req.Size
		// Size will be determined during upload (-1 means unknown)
		actualSize = -1
		isCompressed = true
	}

	opts := minio.PutObjectOptions{
		ContentType: req.ContentType,
	}

	if req.Metadata != nil {
		opts.UserMetadata = req.Metadata
	} else {
		opts.UserMetadata = make(map[string]string)
	}

	// Preserve original name in metadata
	if req.OriginalName != "" {
		opts.UserMetadata["original-name"] = req.OriginalName
	}

	// Add compression metadata
	if isCompressed {
		opts.UserMetadata["compression"] = "zstd"
		opts.UserMetadata["uncompressed-size"] = fmt.Sprintf("%d", uncompressedSize)
	}

	info, err := m.minioClient.PutObject(ctx, req.BucketName, req.ObjectName, reader, actualSize, opts)
	if err != nil {
		return nil, handleMinIOError(err, "upload_file")
	}

	fileInfo := &FileInfo{
		BucketName:       req.BucketName,
		ObjectName:       req.ObjectName,
		OriginalName:     req.OriginalName,
		Size:             info.Size,
		ContentType:      req.ContentType,
		ETag:             info.ETag,
		LastModified:     time.Now(),
		Metadata:         req.Metadata,
		IsCompressed:     isCompressed,
		CompressedSize:   info.Size,
		UncompressedSize: uncompressedSize,
	}

	if isCompressed && uncompressedSize > 0 {
		fileInfo.CompressionRatio = float64(info.Size) / float64(uncompressedSize)
	}

	return fileInfo, nil
}

// GetPresignedUploadURL generates a presigned URL for direct upload.
func (m *implMinIO) GetPresignedUploadURL(ctx context.Context, req *PresignedURLRequest) (*PresignedURLResponse, error) {
	if err := validatePresignedURLRequest(req); err != nil {
		return nil, err
	}

	url, err := m.minioClient.PresignedPutObject(ctx, req.BucketName, req.ObjectName, req.Expiry)
	if err != nil {
		return nil, handleMinIOError(err, "get_presigned_upload_url")
	}

	return &PresignedURLResponse{
		URL:       url.String(),
		ExpiresAt: time.Now().Add(req.Expiry),
		Method:    "PUT",
		Headers:   req.Headers,
	}, nil
}

// DownloadFile downloads a file from MinIO storage.
func (m *implMinIO) DownloadFile(ctx context.Context, req *DownloadRequest) (io.ReadCloser, *DownloadHeaders, error) {
	if err := validateDownloadRequest(req); err != nil {
		return nil, nil, err
	}

	// Get file info for headers
	objInfo, err := m.minioClient.StatObject(ctx, req.BucketName, req.ObjectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, nil, handleMinIOError(err, "get_file_info")
	}

	// Prepare download options
	opts := minio.GetObjectOptions{}
	if req.Range != nil {
		opts.SetRange(req.Range.Start, req.Range.End)
	}

	// Download object
	object, err := m.minioClient.GetObject(ctx, req.BucketName, req.ObjectName, opts)
	if err != nil {
		return nil, nil, handleMinIOError(err, "download_file")
	}

	// Auto-decompress if file is compressed
	var reader io.ReadCloser = object
	if compression, exists := objInfo.UserMetadata["compression"]; exists && compression == "zstd" {
		decompressedReader, err := compressor.DecompressStream(object)
		if err != nil {
			object.Close()
			return nil, nil, fmt.Errorf("decompression failed: %w", err)
		}
		reader = decompressedReader
	}

	// Generate headers
	headers := m.generateDownloadHeaders(objInfo, req)

	return reader, headers, nil
}

// StreamFile streams a file for web viewing (optimized for streaming).
func (m *implMinIO) StreamFile(ctx context.Context, req *DownloadRequest) (io.ReadCloser, *DownloadHeaders, error) {
	// Force inline disposition for streaming
	req.Disposition = "inline"

	reader, headers, err := m.DownloadFile(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	// Override headers for streaming optimization
	headers.CacheControl = "public, max-age=86400"
	headers.AcceptRanges = "bytes"

	if req.Range != nil {
		headers.ContentRange = fmt.Sprintf("bytes %d-%d/%s", req.Range.Start, req.Range.End, headers.ContentLength)
	}

	return reader, headers, nil
}

// GetPresignedDownloadURL generates a presigned URL for direct download.
func (m *implMinIO) GetPresignedDownloadURL(ctx context.Context, req *PresignedURLRequest) (*PresignedURLResponse, error) {
	if err := validatePresignedURLRequest(req); err != nil {
		return nil, err
	}

	url, err := m.minioClient.PresignedGetObject(ctx, req.BucketName, req.ObjectName, req.Expiry, nil)
	if err != nil {
		return nil, handleMinIOError(err, "get_presigned_download_url")
	}

	return &PresignedURLResponse{
		URL:       url.String(),
		ExpiresAt: time.Now().Add(req.Expiry),
		Method:    "GET",
		Headers:   req.Headers,
	}, nil
}

// GetFileInfo retrieves metadata about a file.
func (m *implMinIO) GetFileInfo(ctx context.Context, bucketName, objectName string) (*FileInfo, error) {
	if err := validateBucketName(bucketName); err != nil {
		return nil, err
	}
	if err := validateObjectName(objectName); err != nil {
		return nil, err
	}

	objInfo, err := m.minioClient.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, handleMinIOError(err, "get_file_info")
	}

	fileInfo := &FileInfo{
		BucketName:   bucketName,
		ObjectName:   objectName,
		Size:         objInfo.Size,
		ContentType:  objInfo.ContentType,
		ETag:         objInfo.ETag,
		LastModified: objInfo.LastModified,
		Metadata:     objInfo.UserMetadata,
	}

	if originalName, exists := objInfo.UserMetadata["original-name"]; exists {
		fileInfo.OriginalName = originalName
	}

	return fileInfo, nil
}

// DeleteFile removes a file from storage.
func (m *implMinIO) DeleteFile(ctx context.Context, bucketName, objectName string) error {
	if err := validateBucketName(bucketName); err != nil {
		return err
	}
	if err := validateObjectName(objectName); err != nil {
		return err
	}

	err := m.minioClient.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return handleMinIOError(err, "delete_file")
	}
	return nil
}

// CopyFile copies a file from one location to another.
func (m *implMinIO) CopyFile(ctx context.Context, srcBucket, srcObject, destBucket, destObject string) error {
	srcOpts := minio.CopySrcOptions{
		Bucket: srcBucket,
		Object: srcObject,
	}

	destOpts := minio.CopyDestOptions{
		Bucket: destBucket,
		Object: destObject,
	}

	_, err := m.minioClient.CopyObject(ctx, destOpts, srcOpts)
	if err != nil {
		return handleMinIOError(err, "copy_file")
	}
	return nil
}

// MoveFile moves a file from one location to another (copy + delete).
// If the delete operation fails, it attempts to cleanup the copied file.
func (m *implMinIO) MoveFile(ctx context.Context, srcBucket, srcObject, destBucket, destObject string) error {
	// Copy file first
	if err := m.CopyFile(ctx, srcBucket, srcObject, destBucket, destObject); err != nil {
		return err
	}

	// Delete source file
	if err := m.DeleteFile(ctx, srcBucket, srcObject); err != nil {
		// If delete fails, try to cleanup the copied file
		if cleanupErr := m.DeleteFile(ctx, destBucket, destObject); cleanupErr != nil {
			return fmt.Errorf("move failed: %w, cleanup also failed: %v", err, cleanupErr)
		}
		return fmt.Errorf("move failed: %w", err)
	}

	return nil
}

// FileExists checks if a file exists.
func (m *implMinIO) FileExists(ctx context.Context, bucketName, objectName string) (bool, error) {
	_, err := m.GetFileInfo(ctx, bucketName, objectName)
	if err != nil {
		if storageErr, ok := err.(*StorageError); ok && storageErr.Code == ErrCodeObjectNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListFiles lists files in a bucket with optional filtering.
func (m *implMinIO) ListFiles(ctx context.Context, req *ListRequest) (*ListResponse, error) {
	if err := validateListRequest(req); err != nil {
		return nil, err
	}

	opts := minio.ListObjectsOptions{
		Prefix:    req.Prefix,
		Recursive: req.Recursive,
		MaxKeys:   req.MaxKeys,
	}

	var files []*FileInfo
	objectCh := m.minioClient.ListObjects(ctx, req.BucketName, opts)

	for object := range objectCh {
		if object.Err != nil {
			return nil, handleMinIOError(object.Err, "list_files")
		}

		fileInfo := &FileInfo{
			BucketName:   req.BucketName,
			ObjectName:   object.Key,
			Size:         object.Size,
			ETag:         object.ETag,
			LastModified: object.LastModified,
			ContentType:  object.ContentType,
		}

		files = append(files, fileInfo)
	}

	response := &ListResponse{
		Files:       files,
		TotalCount:  len(files),
		IsTruncated: len(files) >= req.MaxKeys,
	}

	if response.IsTruncated && len(files) > 0 {
		response.NextMarker = files[len(files)-1].ObjectName
	}

	return response, nil
}

// UpdateMetadata updates the metadata of a file.
func (m *implMinIO) UpdateMetadata(ctx context.Context, bucketName, objectName string, metadata map[string]string) error {
	srcOpts := minio.CopySrcOptions{
		Bucket: bucketName,
		Object: objectName,
	}

	destOpts := minio.CopyDestOptions{
		Bucket:          bucketName,
		Object:          objectName,
		UserMetadata:    metadata,
		ReplaceMetadata: true,
	}

	_, err := m.minioClient.CopyObject(ctx, destOpts, srcOpts)
	if err != nil {
		return handleMinIOError(err, "update_metadata")
	}
	return nil
}

// GetMetadata retrieves the metadata of a file.
func (m *implMinIO) GetMetadata(ctx context.Context, bucketName, objectName string) (map[string]string, error) {
	fileInfo, err := m.GetFileInfo(ctx, bucketName, objectName)
	if err != nil {
		return nil, err
	}
	return fileInfo.Metadata, nil
}

// handleMinIOError handles MinIO errors consistently and converts them to StorageError.
func handleMinIOError(err error, operation string) *StorageError {
	if err == nil {
		return nil
	}

	if minioErr, ok := err.(minio.ErrorResponse); ok {
		switch minioErr.Code {
		case "NoSuchBucket":
			return NewBucketNotFoundError("")
		case "NoSuchKey":
			return NewObjectNotFoundError("")
		case "AccessDenied":
			return &StorageError{
				Code:      ErrCodePermission,
				Message:   "Access denied",
				Operation: operation,
				Cause:     err,
			}
		default:
			return &StorageError{
				Code:      ErrCodeConnection,
				Message:   fmt.Sprintf("MinIO operation failed: %s", minioErr.Code),
				Operation: operation,
				Cause:     err,
			}
		}
	}

	return NewConnectionError(err)
}
