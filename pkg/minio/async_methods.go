package minio

import (
	"context"
	"time"
)

// UploadAsync queues an upload task and returns task ID
func (m *implMinIO) UploadAsync(ctx context.Context, req *UploadRequest) (string, error) {
	return m.asyncUploadMgr.UploadAsync(ctx, req)
}

// GetUploadStatus retrieves upload status by task ID
func (m *implMinIO) GetUploadStatus(taskID string) (*UploadProgress, error) {
	return m.asyncUploadMgr.GetUploadStatus(taskID)
}

// WaitForUpload waits for upload completion with timeout
func (m *implMinIO) WaitForUpload(taskID string, timeout time.Duration) (*AsyncUploadResult, error) {
	return m.asyncUploadMgr.WaitForUpload(taskID, timeout)
}

// CancelUpload cancels an upload task
func (m *implMinIO) CancelUpload(taskID string) error {
	return m.asyncUploadMgr.CancelUpload(taskID)
}
