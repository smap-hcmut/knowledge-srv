package model

import (
	"time"

	"github.com/aarondl/null/v8"

	"knowledge-srv/internal/sqlboiler"
)

// Status constants
const (
	StatusPending    = "PENDING"
	StatusIndexed    = "INDEXED"
	StatusFailed     = "FAILED"
	StatusReIndexing = "RE_INDEXING"
)

// Error type constants
const (
	ErrorTypeValidation = "VALIDATION_ERROR"
	ErrorTypeParse      = "PARSE_ERROR"
	ErrorTypeEmbedding  = "EMBEDDING_ERROR"
	ErrorTypeQdrant     = "QDRANT_ERROR"
	ErrorTypeDB         = "DB_ERROR"
	ErrorTypeDuplicate  = "DUPLICATE_CONTENT"
)

// IndexedDocument represents a document tracking record.
type IndexedDocument struct {
	ID          string `json:"id"`
	AnalyticsID string `json:"analytics_id"`
	ProjectID   string `json:"project_id"`
	SourceID    string `json:"source_id"`

	// Qdrant Reference
	QdrantPointID  string `json:"qdrant_point_id"`
	CollectionName string `json:"collection_name"`

	// Content Hash (for deduplication)
	ContentHash string `json:"content_hash"`

	// Indexing Status
	Status       string  `json:"status"`
	ErrorMessage *string `json:"error_message,omitempty"`
	RetryCount   int     `json:"retry_count"`

	// Batch Tracking
	BatchID *string `json:"batch_id,omitempty"`

	// Performance Metrics
	EmbeddingTimeMs int `json:"embedding_time_ms,omitempty"`
	UpsertTimeMs    int `json:"upsert_time_ms,omitempty"`
	TotalTimeMs     int `json:"total_time_ms,omitempty"`

	// Timestamps
	IndexedAt *time.Time `json:"indexed_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// NewIndexedDocumentFromDB converts a SQLBoiler IndexedDocument to IndexedDocument
func NewIndexedDocumentFromDB(db *sqlboiler.IndexedDocument) *IndexedDocument {
	if db == nil {
		return nil
	}

	doc := &IndexedDocument{
		ID:             db.ID,
		AnalyticsID:    db.AnalyticsID,
		ProjectID:      db.ProjectID,
		SourceID:       db.SourceID,
		QdrantPointID:  db.QdrantPointID,
		CollectionName: db.CollectionName,
		ContentHash:    db.ContentHash,
		Status:         db.Status,
	}

	// Handle nullable fields
	if db.ErrorMessage.Valid {
		doc.ErrorMessage = &db.ErrorMessage.String
	}
	if db.RetryCount.Valid {
		doc.RetryCount = int(db.RetryCount.Int)
	}
	if db.BatchID.Valid {
		doc.BatchID = &db.BatchID.String
	}
	if db.EmbeddingTimeMS.Valid {
		doc.EmbeddingTimeMs = int(db.EmbeddingTimeMS.Int)
	}
	if db.UpsertTimeMS.Valid {
		doc.UpsertTimeMs = int(db.UpsertTimeMS.Int)
	}
	if db.TotalTimeMS.Valid {
		doc.TotalTimeMs = int(db.TotalTimeMS.Int)
	}
	if db.IndexedAt.Valid {
		doc.IndexedAt = &db.IndexedAt.Time
	}
	if db.CreatedAt.Valid {
		doc.CreatedAt = db.CreatedAt.Time
	}
	if db.UpdatedAt.Valid {
		doc.UpdatedAt = db.UpdatedAt.Time
	}

	return doc
}

// ToDBIndexedDocument converts IndexedDocument to SQLBoiler IndexedDocument
func (d *IndexedDocument) ToDBIndexedDocument() *sqlboiler.IndexedDocument {
	db := &sqlboiler.IndexedDocument{
		ID:             d.ID,
		AnalyticsID:    d.AnalyticsID,
		ProjectID:      d.ProjectID,
		SourceID:       d.SourceID,
		QdrantPointID:  d.QdrantPointID,
		CollectionName: d.CollectionName,
		ContentHash:    d.ContentHash,
		Status:         d.Status,
	}

	// Handle nullable fields
	if d.ErrorMessage != nil {
		db.ErrorMessage = null.StringFrom(*d.ErrorMessage)
	}
	db.RetryCount = null.IntFrom(d.RetryCount)
	if d.BatchID != nil {
		db.BatchID = null.StringFrom(*d.BatchID)
	}
	if d.EmbeddingTimeMs > 0 {
		db.EmbeddingTimeMS = null.IntFrom(d.EmbeddingTimeMs)
	}
	if d.UpsertTimeMs > 0 {
		db.UpsertTimeMS = null.IntFrom(d.UpsertTimeMs)
	}
	if d.TotalTimeMs > 0 {
		db.TotalTimeMS = null.IntFrom(d.TotalTimeMs)
	}
	if d.IndexedAt != nil {
		db.IndexedAt = null.TimeFrom(*d.IndexedAt)
	}
	db.CreatedAt = null.TimeFrom(d.CreatedAt)
	db.UpdatedAt = null.TimeFrom(d.UpdatedAt)

	return db
}
