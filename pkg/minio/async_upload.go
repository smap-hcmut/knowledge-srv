package minio

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// UploadStatus represents the status of an async upload
type UploadStatus string

const (
	UploadStatusPending   UploadStatus = "pending"
	UploadStatusUploading UploadStatus = "uploading"
	UploadStatusCompleted UploadStatus = "completed"
	UploadStatusFailed    UploadStatus = "failed"
	UploadStatusCancelled UploadStatus = "cancelled"
)

// AsyncUploadTask represents a single async upload task
type AsyncUploadTask struct {
	ID           string
	Request      *UploadRequest
	ResultChan   chan *AsyncUploadResult
	ProgressChan chan *UploadProgress
	CreatedAt    time.Time
	ctx          context.Context
	cancel       context.CancelFunc
}

// AsyncUploadResult contains the result of an async upload
type AsyncUploadResult struct {
	TaskID    string
	FileInfo  *FileInfo
	Error     error
	Duration  time.Duration
	StartTime time.Time
	EndTime   time.Time
}

// UploadProgress represents the progress of an upload
type UploadProgress struct {
	TaskID        string       `json:"task_id"`
	BytesUploaded int64        `json:"bytes_uploaded"`
	TotalBytes    int64        `json:"total_bytes"`
	Percentage    float64      `json:"percentage"`
	Status        UploadStatus `json:"status"`
	Error         string       `json:"error,omitempty"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// AsyncUploadManager manages async upload operations
type AsyncUploadManager struct {
	minio         *implMinIO
	workerPool    int
	uploadQueue   chan *AsyncUploadTask
	statusTracker *UploadStatusTracker
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	started       bool
	mu            sync.RWMutex
}

// NewAsyncUploadManager creates a new async upload manager
func NewAsyncUploadManager(minio *implMinIO, workerPoolSize int, queueSize int) *AsyncUploadManager {
	if workerPoolSize <= 0 {
		workerPoolSize = 4 // default
	}
	if queueSize <= 0 {
		queueSize = 100 // default
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &AsyncUploadManager{
		minio:         minio,
		workerPool:    workerPoolSize,
		uploadQueue:   make(chan *AsyncUploadTask, queueSize),
		statusTracker: NewUploadStatusTracker(),
		ctx:           ctx,
		cancel:        cancel,
		started:       false,
	}
}

// Start starts the worker pool
func (m *AsyncUploadManager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return
	}

	// Start workers
	for i := 0; i < m.workerPool; i++ {
		m.wg.Add(1)
		go m.worker(i)
	}

	// Start cleanup goroutine
	m.wg.Add(1)
	go m.cleanupWorker()

	m.started = true
}

// Stop gracefully stops the worker pool
func (m *AsyncUploadManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return
	}

	// Cancel context to stop workers
	m.cancel()

	// Close upload queue
	close(m.uploadQueue)

	// Wait for all workers to finish
	m.wg.Wait()

	m.started = false
}

// UploadAsync queues an upload task and returns task ID
func (m *AsyncUploadManager) UploadAsync(ctx context.Context, req *UploadRequest) (string, error) {
	m.mu.RLock()
	if !m.started {
		m.mu.RUnlock()
		return "", fmt.Errorf("async upload manager not started")
	}
	m.mu.RUnlock()

	// Generate task ID
	taskID := uuid.New().String()

	// Create task context
	taskCtx, taskCancel := context.WithCancel(ctx)

	// Create task
	task := &AsyncUploadTask{
		ID:           taskID,
		Request:      req,
		ResultChan:   make(chan *AsyncUploadResult, 1),
		ProgressChan: make(chan *UploadProgress, 10),
		CreatedAt:    time.Now(),
		ctx:          taskCtx,
		cancel:       taskCancel,
	}

	// Initialize status
	m.statusTracker.UpdateStatus(taskID, &UploadProgress{
		TaskID:     taskID,
		TotalBytes: req.Size,
		Status:     UploadStatusPending,
		UpdatedAt:  time.Now(),
	})

	// Queue task
	select {
	case m.uploadQueue <- task:
		return taskID, nil
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		return "", fmt.Errorf("upload queue is full")
	}
}

// GetUploadStatus retrieves upload status by task ID
func (m *AsyncUploadManager) GetUploadStatus(taskID string) (*UploadProgress, error) {
	progress, exists := m.statusTracker.GetStatus(taskID)
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}
	return progress, nil
}

// WaitForUpload waits for upload completion with timeout
func (m *AsyncUploadManager) WaitForUpload(taskID string, timeout time.Duration) (*AsyncUploadResult, error) {
	// Get task from tracker
	progress, exists := m.statusTracker.GetStatus(taskID)
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	// If already completed or failed, return immediately
	if progress.Status == UploadStatusCompleted || progress.Status == UploadStatusFailed {
		// Try to get result from tracker
		result := m.statusTracker.GetResult(taskID)
		if result != nil {
			return result, nil
		}
		return nil, fmt.Errorf("task %s is %s but result not available", taskID, progress.Status)
	}

	// Wait for completion with timeout
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	for {
		select {
		case <-timeoutTimer.C:
			return nil, fmt.Errorf("timeout waiting for upload: %s", taskID)
		case <-ticker.C:
			progress, exists := m.statusTracker.GetStatus(taskID)
			if !exists {
				return nil, fmt.Errorf("task disappeared: %s", taskID)
			}

			if progress.Status == UploadStatusCompleted || progress.Status == UploadStatusFailed {
				result := m.statusTracker.GetResult(taskID)
				if result != nil {
					return result, nil
				}
				return nil, fmt.Errorf("task %s is %s but result not available", taskID, progress.Status)
			}
		}
	}
}

// CancelUpload cancels an upload task
func (m *AsyncUploadManager) CancelUpload(taskID string) error {
	progress, exists := m.statusTracker.GetStatus(taskID)
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Can only cancel pending or uploading tasks
	if progress.Status != UploadStatusPending && progress.Status != UploadStatusUploading {
		return fmt.Errorf("cannot cancel task in status: %s", progress.Status)
	}

	// Update status to cancelled
	m.statusTracker.UpdateStatus(taskID, &UploadProgress{
		TaskID:    taskID,
		Status:    UploadStatusCancelled,
		UpdatedAt: time.Now(),
	})

	return nil
}

// worker processes upload tasks
func (m *AsyncUploadManager) worker(workerID int) {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return
		case task, ok := <-m.uploadQueue:
			if !ok {
				return
			}
			m.processUploadTask(workerID, task)
		}
	}
}

// processUploadTask processes a single upload task
func (m *AsyncUploadManager) processUploadTask(workerID int, task *AsyncUploadTask) {
	startTime := time.Now()

	// Check if task was cancelled before starting
	progress, _ := m.statusTracker.GetStatus(task.ID)
	if progress != nil && progress.Status == UploadStatusCancelled {
		return
	}

	// Update status to uploading
	m.statusTracker.UpdateStatus(task.ID, &UploadProgress{
		TaskID:     task.ID,
		TotalBytes: task.Request.Size,
		Status:     UploadStatusUploading,
		UpdatedAt:  time.Now(),
	})

	// Create progress reader wrapper
	progressReader := &ProgressReader{
		Reader:     task.Request.Reader,
		TotalBytes: task.Request.Size,
		OnProgress: func(bytesRead int64) {
			// Check if cancelled
			progress, _ := m.statusTracker.GetStatus(task.ID)
			if progress != nil && progress.Status == UploadStatusCancelled {
				return
			}

			percentage := float64(bytesRead) / float64(task.Request.Size) * 100
			if percentage > 100 {
				percentage = 100
			}

			m.statusTracker.UpdateStatus(task.ID, &UploadProgress{
				TaskID:        task.ID,
				BytesUploaded: bytesRead,
				TotalBytes:    task.Request.Size,
				Percentage:    percentage,
				Status:        UploadStatusUploading,
				UpdatedAt:     time.Now(),
			})
		},
	}

	// Replace reader with progress reader
	task.Request.Reader = progressReader

	// Perform upload
	fileInfo, err := m.minio.UploadFile(task.ctx, task.Request)

	duration := time.Since(startTime)
	endTime := time.Now()

	result := &AsyncUploadResult{
		TaskID:    task.ID,
		FileInfo:  fileInfo,
		Error:     err,
		Duration:  duration,
		StartTime: startTime,
		EndTime:   endTime,
	}

	// Update final status
	if err != nil {
		m.statusTracker.UpdateStatus(task.ID, &UploadProgress{
			TaskID:    task.ID,
			Status:    UploadStatusFailed,
			Error:     err.Error(),
			UpdatedAt: time.Now(),
		})
	} else {
		m.statusTracker.UpdateStatus(task.ID, &UploadProgress{
			TaskID:        task.ID,
			BytesUploaded: task.Request.Size,
			TotalBytes:    task.Request.Size,
			Percentage:    100,
			Status:        UploadStatusCompleted,
			UpdatedAt:     time.Now(),
		})
	}

	// Store result
	m.statusTracker.StoreResult(task.ID, result)

	// Send result to channel (non-blocking)
	select {
	case task.ResultChan <- result:
	default:
	}
}

// cleanupWorker periodically cleans up old completed/failed tasks
func (m *AsyncUploadManager) cleanupWorker() {
	defer m.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			// Cleanup tasks older than 1 hour
			m.statusTracker.CleanupOldStatuses(1 * time.Hour)
		}
	}
}
