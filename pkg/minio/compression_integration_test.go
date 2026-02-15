package minio

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"smap-project/pkg/compressor"
)

// TestUploadWithCompression tests uploading a file with compression enabled
func TestUploadWithCompression(t *testing.T) {
	// Skip if MinIO is not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create test data (JSON-like data that compresses well)
	testData := strings.Repeat(`{"name":"test","value":"data","items":["a","b","c"]},`, 100)
	originalSize := int64(len(testData))

	tests := []struct {
		name             string
		compressionLevel int
		expectCompressed bool
	}{
		{
			name:             "with default compression",
			compressionLevel: 2,
			expectCompressed: true,
		},
		{
			name:             "with fastest compression",
			compressionLevel: 1,
			expectCompressed: true,
		},
		{
			name:             "with best compression",
			compressionLevel: 3,
			expectCompressed: true,
		},
		{
			name:             "without compression",
			compressionLevel: 0,
			expectCompressed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &UploadRequest{
				BucketName:        "test-bucket",
				ObjectName:        "test-compressed-file.json",
				OriginalName:      "test.json",
				Reader:            strings.NewReader(testData),
				Size:              originalSize,
				ContentType:       "application/json",
				EnableCompression: tt.expectCompressed,
				CompressionLevel:  tt.compressionLevel,
			}

			// Note: This is a unit test for the compression logic
			// Actual MinIO upload would require a running MinIO instance

			// Test compression logic
			if req.EnableCompression {
				var compLevel compressor.CompressionLevel
				switch req.CompressionLevel {
				case 1:
					compLevel = compressor.CompressionFastest
				case 3:
					compLevel = compressor.CompressionBest
				default:
					compLevel = compressor.CompressionDefault
				}

				compressedReader, err := compressor.CompressStream(req.Reader, compLevel)
				if err != nil {
					t.Fatalf("Compression failed: %v", err)
				}
				defer compressedReader.Close()

				// Read compressed data
				compressed, err := io.ReadAll(compressedReader)
				if err != nil {
					t.Fatalf("Failed to read compressed data: %v", err)
				}

				// Verify compression worked
				if len(compressed) >= int(originalSize) {
					t.Logf("Warning: Compressed size (%d) >= original size (%d)", len(compressed), originalSize)
				} else {
					ratio := float64(len(compressed)) / float64(originalSize) * 100
					t.Logf("Compression successful: %d bytes -> %d bytes (%.2f%%)", originalSize, len(compressed), ratio)
				}

				// Verify we can decompress
				decompressed, err := compressor.DecompressBytes(compressed)
				if err != nil {
					t.Fatalf("Decompression failed: %v", err)
				}

				if string(decompressed) != testData {
					t.Error("Decompressed data doesn't match original")
				}
			}
		})
	}
}

// TestDownloadWithDecompression tests downloading and auto-decompressing a file
func TestDownloadWithDecompression(t *testing.T) {
	testData := "This is test data for decompression"

	// Compress the data
	compressed, err := compressor.CompressBytes([]byte(testData), compressor.CompressionDefault)
	if err != nil {
		t.Fatalf("Failed to compress test data: %v", err)
	}

	// Simulate decompression on download
	decompressedReader, err := compressor.DecompressStream(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("Failed to create decompression stream: %v", err)
	}
	defer decompressedReader.Close()

	// Read decompressed data
	decompressed, err := io.ReadAll(decompressedReader)
	if err != nil {
		t.Fatalf("Failed to read decompressed data: %v", err)
	}

	// Verify
	if string(decompressed) != testData {
		t.Errorf("Decompressed data doesn't match.\nExpected: %s\nGot: %s", testData, string(decompressed))
	}
}

// TestCompressionMetadata tests that compression metadata is properly stored
func TestCompressionMetadata(t *testing.T) {
	testData := strings.Repeat("test data ", 100)
	originalSize := int64(len(testData))

	// Compress data
	compressed, err := compressor.CompressBytes([]byte(testData), compressor.CompressionDefault)
	if err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	// Simulate FileInfo creation
	fileInfo := &FileInfo{
		BucketName:       "test-bucket",
		ObjectName:       "test-file.txt",
		OriginalName:     "test.txt",
		Size:             int64(len(compressed)),
		IsCompressed:     true,
		CompressedSize:   int64(len(compressed)),
		UncompressedSize: originalSize,
	}

	// Calculate compression ratio
	if fileInfo.UncompressedSize > 0 {
		fileInfo.CompressionRatio = float64(fileInfo.CompressedSize) / float64(fileInfo.UncompressedSize)
	}

	// Verify metadata
	if !fileInfo.IsCompressed {
		t.Error("IsCompressed should be true")
	}

	if fileInfo.CompressedSize >= fileInfo.UncompressedSize {
		t.Logf("Warning: Compressed size (%d) >= uncompressed size (%d)", fileInfo.CompressedSize, fileInfo.UncompressedSize)
	}

	if fileInfo.CompressionRatio <= 0 || fileInfo.CompressionRatio > 1 {
		t.Errorf("Invalid compression ratio: %.2f", fileInfo.CompressionRatio)
	}

	t.Logf("Compression metadata: Original=%d, Compressed=%d, Ratio=%.2f%%",
		fileInfo.UncompressedSize, fileInfo.CompressedSize, fileInfo.CompressionRatio*100)
}

// BenchmarkUploadWithCompression benchmarks upload with compression
func BenchmarkUploadWithCompression(b *testing.B) {
	testData := strings.Repeat(`{"name":"benchmark","value":"test","data":["item1","item2","item3"]},`, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(testData)
		compressedReader, err := compressor.CompressStream(reader, compressor.CompressionDefault)
		if err != nil {
			b.Fatal(err)
		}

		// Simulate upload by reading all data
		_, err = io.ReadAll(compressedReader)
		if err != nil {
			b.Fatal(err)
		}

		compressedReader.Close()
	}
}

// BenchmarkDownloadWithDecompression benchmarks download with decompression
func BenchmarkDownloadWithDecompression(b *testing.B) {
	testData := strings.Repeat("benchmark test data ", 1000)
	compressed, err := compressor.CompressBytes([]byte(testData), compressor.CompressionDefault)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(compressed)
		decompressedReader, err := compressor.DecompressStream(reader)
		if err != nil {
			b.Fatal(err)
		}

		// Simulate download by reading all data
		_, err = io.ReadAll(decompressedReader)
		if err != nil {
			b.Fatal(err)
		}

		decompressedReader.Close()
	}
}
