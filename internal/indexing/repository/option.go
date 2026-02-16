package repository

import "time"

// =====================================================
// IndexedDocument Options
// =====================================================

// CreateDocumentOptions - Options for Create operation
type CreateDocumentOptions struct {
	AnalyticsID     string
	ProjectID       string
	SourceID        string
	QdrantPointID   string
	CollectionName  string
	ContentHash     string
	Status          string
	ErrorMessage    *string
	RetryCount      int
	BatchID         *string
	EmbeddingTimeMs int
	UpsertTimeMs    int
	TotalTimeMs     int
	IndexedAt       *time.Time
}

// UpsertDocumentOptions - Options for Upsert operation
type UpsertDocumentOptions struct {
	AnalyticsID     string
	ProjectID       string
	SourceID        string
	QdrantPointID   string
	CollectionName  string
	ContentHash     string
	Status          string
	ErrorMessage    *string
	RetryCount      int
	BatchID         *string
	EmbeddingTimeMs int
	UpsertTimeMs    int
	TotalTimeMs     int
	IndexedAt       *time.Time
}

// GetOneDocumentOptions - Options for GetOne query (single record by filters)
// If multiple filters are provided, they will be combined with AND condition
type GetOneDocumentOptions struct {
	AnalyticsID string // Filter by analytics_id
	ContentHash string // Filter by content_hash
}

// GetDocumentsOptions - Options for Get query (with pagination)
type GetDocumentsOptions struct {
	// Filters
	Status      string     // Filter by status (PENDING, INDEXED, FAILED, RE_INDEXING)
	ProjectID   string     // Filter by project_id
	BatchID     string     // Filter by batch_id
	ErrorTypes  []string   // Filter by error types (for retry)
	MaxRetry    int        // Filter retry_count < max (for retry logic)
	StaleBefore *time.Time // Filter created_at < stale_before (for reconcile)

	// Pagination (REQUIRED for Get)
	Limit  int // Number of records per page
	Offset int // Number of records to skip

	// Sorting
	OrderBy string // e.g., "created_at DESC", "indexed_at ASC"
}

// ListDocumentsOptions - Options for List query (without pagination)
type ListDocumentsOptions struct {
	// Filters
	Status      string     // Filter by status
	ProjectID   string     // Filter by project_id
	BatchID     string     // Filter by batch_id
	ErrorTypes  []string   // Filter by error types
	MaxRetry    int        // Filter retry_count < max
	StaleBefore *time.Time // Filter created_at < stale_before

	// Optional Safety Limit (to prevent accidental full table scans)
	Limit int // Max records to return (0 = no limit, but cap at 10000 for safety)

	// Sorting
	OrderBy string // e.g., "created_at ASC"
}

// UpdateDocumentStatusOptions - Options when updating status
type UpdateDocumentStatusOptions struct {
	ID      string
	Status  string
	Metrics DocumentStatusMetrics
}

// DocumentStatusMetrics - Metrics when updating status
type DocumentStatusMetrics struct {
	IndexedAt       *time.Time // When successfully indexed
	ErrorMessage    string     // Error message if failed
	RetryCount      int        // Number of retries
	EmbeddingTimeMs int        // Time spent on embedding (ms)
	UpsertTimeMs    int        // Time spent on Qdrant upsert (ms)
	TotalTimeMs     int        // Total processing time (ms)
}

// ProjectStats - Statistics per project
type DocumentProjectStats struct {
	ProjectID      string
	TotalIndexed   int        // Count of INDEXED records
	TotalFailed    int        // Count of FAILED records
	TotalPending   int        // Count of PENDING records
	LastIndexedAt  *time.Time // Most recent indexed_at
	AvgIndexTimeMs int        // Average total_time_ms for INDEXED records
}

// =====================================================
// DLQ Options
// =====================================================

// CreateDLQOptions - Options for CreateDLQ operation
type CreateDLQOptions struct {
	AnalyticsID  string
	ProjectID    string
	SourceID     string
	ContentHash  string
	ErrorType    string
	ErrorMessage string
	RetryCount   int
	BatchID      *string
	FailedAt     time.Time
}

// GetOneDLQOptions - Options for GetOneDLQ query
type GetOneDLQOptions struct {
	ID          string // Filter by id
	AnalyticsID string // Filter by analytics_id
	ContentHash string // Filter by content_hash
}

// ListDLQOptions - Options for ListDLQ query (no pagination)
type ListDLQOptions struct {
	// Filters
	ProjectID      string   // Filter by project_id
	ErrorTypes     []string // Filter by error types
	ResolvedOnly   bool     // Filter resolved_at IS NOT NULL
	UnresolvedOnly bool     // Filter resolved_at IS NULL

	// Optional Safety Limit
	Limit int // Max records to return (0 = no limit)

	// Sorting
	OrderBy string // e.g., "created_at DESC"
}

// UpsertPointOptions - Options for UpsertPoint operation
type UpsertPointOptions struct {
	PointID string
	Vector  []float32
	Payload map[string]interface{}
}
