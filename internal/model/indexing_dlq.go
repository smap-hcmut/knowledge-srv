package model

import (
	"knowledge-srv/internal/sqlboiler"
	"time"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/types"
)

// IndexingDLQ represents a dead letter queue record.
type IndexingDLQ struct {
	ID          string  `json:"id"`
	AnalyticsID string  `json:"analytics_id"`
	BatchID     *string `json:"batch_id,omitempty"`

	// Error Details
	RawPayload   []byte `json:"raw_payload"` // JSON bytes
	ErrorMessage string `json:"error_message"`
	ErrorType    string `json:"error_type"`

	// Retry Management
	RetryCount int `json:"retry_count"`
	MaxRetries int `json:"max_retries"`

	// Resolution Status
	Resolved bool `json:"resolved"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewIndexingDLQFromDB converts a SQLBoiler IndexingDLQ to IndexingDLQ
func NewIndexingDLQFromDB(db *sqlboiler.IndexingDLQ) *IndexingDLQ {
	if db == nil {
		return nil
	}

	dlq := &IndexingDLQ{
		ID:           db.ID,
		AnalyticsID:  db.AnalyticsID,
		ErrorMessage: db.ErrorMessage,
		ErrorType:    db.ErrorType,
	}

	// Handle nullable fields
	if db.BatchID.Valid {
		dlq.BatchID = &db.BatchID.String
	}
	if db.RawPayload != nil {
		dlq.RawPayload = db.RawPayload
	}
	if db.RetryCount.Valid {
		dlq.RetryCount = int(db.RetryCount.Int)
	}
	if db.MaxRetries.Valid {
		dlq.MaxRetries = int(db.MaxRetries.Int)
	}
	if db.Resolved.Valid {
		dlq.Resolved = db.Resolved.Bool
	}
	if db.CreatedAt.Valid {
		dlq.CreatedAt = db.CreatedAt.Time
	}
	if db.UpdatedAt.Valid {
		dlq.UpdatedAt = db.UpdatedAt.Time
	}

	return dlq
}

// ToDBIndexingDLQ converts IndexingDLQ to SQLBoiler IndexingDLQ
func (d *IndexingDLQ) ToDBIndexingDLQ() *sqlboiler.IndexingDLQ {
	db := &sqlboiler.IndexingDLQ{
		ID:           d.ID,
		AnalyticsID:  d.AnalyticsID,
		ErrorMessage: d.ErrorMessage,
		ErrorType:    d.ErrorType,
	}

	// Handle nullable fields
	if d.BatchID != nil {
		db.BatchID = null.StringFrom(*d.BatchID)
	}
	if d.RawPayload != nil {
		db.RawPayload = types.JSON(d.RawPayload)
	}
	db.RetryCount = null.IntFrom(d.RetryCount)
	db.MaxRetries = null.IntFrom(d.MaxRetries)
	db.Resolved = null.BoolFrom(d.Resolved)
	db.CreatedAt = null.TimeFrom(d.CreatedAt)
	db.UpdatedAt = null.TimeFrom(d.UpdatedAt)

	return db
}
