package minio

import (
	"io"
	"sync"
)

// ProgressReader wraps an io.Reader to track upload progress
type ProgressReader struct {
	Reader     io.Reader
	TotalBytes int64
	bytesRead  int64
	OnProgress func(bytesRead int64)
	mu         sync.Mutex
}

// Read implements io.Reader interface
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)

	if n > 0 {
		pr.mu.Lock()
		pr.bytesRead += int64(n)
		bytesRead := pr.bytesRead
		pr.mu.Unlock()

		if pr.OnProgress != nil {
			pr.OnProgress(bytesRead)
		}
	}

	return n, err
}

// BytesRead returns the total bytes read so far
func (pr *ProgressReader) BytesRead() int64 {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	return pr.bytesRead
}

// Progress returns the current progress percentage
func (pr *ProgressReader) Progress() float64 {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if pr.TotalBytes == 0 {
		return 0
	}

	return float64(pr.bytesRead) / float64(pr.TotalBytes) * 100
}
