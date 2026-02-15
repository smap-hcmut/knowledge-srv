package minio

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"
)

func TestAsyncUploadManager(t *testing.T) {
	// Create a mock MinIO implementation
	// Note: This test focuses on the async upload manager logic
	// Actual MinIO upload would require a running MinIO instance

	t.Run("upload status tracking", func(t *testing.T) {
		tracker := NewUploadStatusTracker()

		taskID := "test-task-1"
		progress := &UploadProgress{
			TaskID:     taskID,
			TotalBytes: 1000,
			Status:     UploadStatusPending,
			UpdatedAt:  time.Now(),
		}

		// Update status
		tracker.UpdateStatus(taskID, progress)

		// Get status
		retrieved, exists := tracker.GetStatus(taskID)
		if !exists {
			t.Fatal("Status should exist")
		}

		if retrieved.TaskID != taskID {
			t.Errorf("TaskID mismatch: got %s, want %s", retrieved.TaskID, taskID)
		}

		if retrieved.Status != UploadStatusPending {
			t.Errorf("Status mismatch: got %s, want %s", retrieved.Status, UploadStatusPending)
		}

		// Update progress
		tracker.UpdateStatus(taskID, &UploadProgress{
			TaskID:        taskID,
			BytesUploaded: 500,
			Percentage:    50,
			Status:        UploadStatusUploading,
			UpdatedAt:     time.Now(),
		})

		retrieved, _ = tracker.GetStatus(taskID)
		if retrieved.BytesUploaded != 500 {
			t.Errorf("BytesUploaded mismatch: got %d, want 500", retrieved.BytesUploaded)
		}

		if retrieved.Percentage != 50 {
			t.Errorf("Percentage mismatch: got %.2f, want 50.00", retrieved.Percentage)
		}
	})

	t.Run("status cleanup", func(t *testing.T) {
		tracker := NewUploadStatusTracker()

		// Add old completed task
		oldTaskID := "old-task"
		tracker.UpdateStatus(oldTaskID, &UploadProgress{
			TaskID:    oldTaskID,
			Status:    UploadStatusCompleted,
			UpdatedAt: time.Now().Add(-2 * time.Hour),
		})

		// Add recent task
		recentTaskID := "recent-task"
		tracker.UpdateStatus(recentTaskID, &UploadProgress{
			TaskID:    recentTaskID,
			Status:    UploadStatusPending,
			UpdatedAt: time.Now(),
		})

		// Cleanup old tasks (older than 1 hour)
		tracker.CleanupOldStatuses(1 * time.Hour)

		// Old task should be removed
		_, exists := tracker.GetStatus(oldTaskID)
		if exists {
			t.Error("Old task should have been cleaned up")
		}

		// Recent task should still exist
		_, exists = tracker.GetStatus(recentTaskID)
		if !exists {
			t.Error("Recent task should still exist")
		}
	})

	t.Run("status counts", func(t *testing.T) {
		tracker := NewUploadStatusTracker()

		// Add tasks with different statuses
		tracker.UpdateStatus("task-1", &UploadProgress{
			TaskID:    "task-1",
			Status:    UploadStatusPending,
			UpdatedAt: time.Now(),
		})

		tracker.UpdateStatus("task-2", &UploadProgress{
			TaskID:    "task-2",
			Status:    UploadStatusUploading,
			UpdatedAt: time.Now(),
		})

		tracker.UpdateStatus("task-3", &UploadProgress{
			TaskID:    "task-3",
			Status:    UploadStatusCompleted,
			UpdatedAt: time.Now(),
		})

		counts := tracker.Count()

		if counts[UploadStatusPending] != 1 {
			t.Errorf("Pending count: got %d, want 1", counts[UploadStatusPending])
		}

		if counts[UploadStatusUploading] != 1 {
			t.Errorf("Uploading count: got %d, want 1", counts[UploadStatusUploading])
		}

		if counts[UploadStatusCompleted] != 1 {
			t.Errorf("Completed count: got %d, want 1", counts[UploadStatusCompleted])
		}
	})
}

func TestProgressReader(t *testing.T) {
	data := "This is test data for progress tracking"
	reader := strings.NewReader(data)

	var progressUpdates []int64
	progressReader := &ProgressReader{
		Reader:     reader,
		TotalBytes: int64(len(data)),
		OnProgress: func(bytesRead int64) {
			progressUpdates = append(progressUpdates, bytesRead)
		},
	}

	// Read all data
	buf := make([]byte, 10) // Read in chunks
	for {
		n, err := progressReader.Read(buf)
		if n == 0 || err != nil {
			break
		}
	}

	// Verify progress was tracked
	if len(progressUpdates) == 0 {
		t.Error("No progress updates received")
	}

	// Verify final bytes read
	finalBytesRead := progressReader.BytesRead()
	if finalBytesRead != int64(len(data)) {
		t.Errorf("BytesRead: got %d, want %d", finalBytesRead, len(data))
	}

	// Verify progress percentage
	progress := progressReader.Progress()
	if progress != 100 {
		t.Errorf("Progress: got %.2f%%, want 100.00%%", progress)
	}
}

func TestAsyncUploadIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test demonstrates the async upload flow
	// In a real scenario, this would use an actual MinIO instance

	t.Run("async upload flow simulation", func(t *testing.T) {
		// Simulate upload request
		testData := strings.Repeat("test data ", 100)
		req := &UploadRequest{
			BucketName:        "test-bucket",
			ObjectName:        "test-async-file.txt",
			OriginalName:      "test.txt",
			Reader:            bytes.NewReader([]byte(testData)),
			Size:              int64(len(testData)),
			ContentType:       "text/plain",
			EnableCompression: true,
			CompressionLevel:  2,
		}

		// Simulate progress tracking
		progressReader := &ProgressReader{
			Reader:     req.Reader,
			TotalBytes: req.Size,
			OnProgress: func(bytesRead int64) {
				percentage := float64(bytesRead) / float64(req.Size) * 100
				t.Logf("Upload progress: %d/%d bytes (%.2f%%)", bytesRead, req.Size, percentage)
			},
		}

		// Read all data (simulating upload)
		uploaded, err := io.ReadAll(progressReader)
		if err != nil {
			t.Fatalf("Failed to read data: %v", err)
		}

		if len(uploaded) != len(testData) {
			t.Errorf("Uploaded size mismatch: got %d, want %d", len(uploaded), len(testData))
		}

		// Verify progress tracking
		if progressReader.BytesRead() != req.Size {
			t.Errorf("BytesRead mismatch: got %d, want %d", progressReader.BytesRead(), req.Size)
		}

		if progressReader.Progress() != 100 {
			t.Errorf("Progress should be 100%%, got %.2f%%", progressReader.Progress())
		}
	})
}

func BenchmarkAsyncUploadStatusTracking(b *testing.B) {
	tracker := NewUploadStatusTracker()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		taskID := "task-" + string(rune(i))
		tracker.UpdateStatus(taskID, &UploadProgress{
			TaskID:        taskID,
			BytesUploaded: int64(i * 1000),
			TotalBytes:    10000,
			Percentage:    float64(i*1000) / 10000 * 100,
			Status:        UploadStatusUploading,
			UpdatedAt:     time.Now(),
		})
	}
}

func BenchmarkProgressReader(b *testing.B) {
	data := bytes.Repeat([]byte("benchmark data "), 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		progressReader := &ProgressReader{
			Reader:     reader,
			TotalBytes: int64(len(data)),
			OnProgress: func(bytesRead int64) {
				// Simulate progress callback
			},
		}

		// Read all data
		io.ReadAll(progressReader)
	}
}
