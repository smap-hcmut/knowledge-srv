package minio

import (
	"sync"
	"time"
)

// UploadStatusTracker tracks the status of async uploads
type UploadStatusTracker struct {
	statuses map[string]*UploadProgress
	results  map[string]*AsyncUploadResult
	mu       sync.RWMutex
}

// NewUploadStatusTracker creates a new upload status tracker
func NewUploadStatusTracker() *UploadStatusTracker {
	return &UploadStatusTracker{
		statuses: make(map[string]*UploadProgress),
		results:  make(map[string]*AsyncUploadResult),
	}
}

// UpdateStatus updates the status of an upload task
func (t *UploadStatusTracker) UpdateStatus(taskID string, progress *UploadProgress) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Update or create status
	if existing, exists := t.statuses[taskID]; exists {
		// Merge with existing status
		if progress.BytesUploaded > 0 {
			existing.BytesUploaded = progress.BytesUploaded
		}
		if progress.TotalBytes > 0 {
			existing.TotalBytes = progress.TotalBytes
		}
		if progress.Percentage > 0 {
			existing.Percentage = progress.Percentage
		}
		if progress.Status != "" {
			existing.Status = progress.Status
		}
		if progress.Error != "" {
			existing.Error = progress.Error
		}
		existing.UpdatedAt = progress.UpdatedAt
	} else {
		t.statuses[taskID] = progress
	}
}

// GetStatus retrieves the status of an upload task
func (t *UploadStatusTracker) GetStatus(taskID string) (*UploadProgress, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	progress, exists := t.statuses[taskID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	progressCopy := *progress
	return &progressCopy, true
}

// StoreResult stores the result of an upload task
func (t *UploadStatusTracker) StoreResult(taskID string, result *AsyncUploadResult) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.results[taskID] = result
}

// GetResult retrieves the result of an upload task
func (t *UploadStatusTracker) GetResult(taskID string) *AsyncUploadResult {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.results[taskID]
}

// CleanupOldStatuses removes old completed/failed tasks
func (t *UploadStatusTracker) CleanupOldStatuses(maxAge time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for taskID, progress := range t.statuses {
		// Only cleanup completed or failed tasks
		if progress.Status == UploadStatusCompleted || progress.Status == UploadStatusFailed || progress.Status == UploadStatusCancelled {
			if now.Sub(progress.UpdatedAt) > maxAge {
				delete(t.statuses, taskID)
				delete(t.results, taskID)
			}
		}
	}
}

// GetAllStatuses returns all current upload statuses
func (t *UploadStatusTracker) GetAllStatuses() map[string]*UploadProgress {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Return a copy
	statuses := make(map[string]*UploadProgress, len(t.statuses))
	for taskID, progress := range t.statuses {
		progressCopy := *progress
		statuses[taskID] = &progressCopy
	}

	return statuses
}

// Count returns the number of tasks in each status
func (t *UploadStatusTracker) Count() map[UploadStatus]int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	counts := make(map[UploadStatus]int)
	for _, progress := range t.statuses {
		counts[progress.Status]++
	}

	return counts
}
