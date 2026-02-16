package postgre

import (
	"time"

	"github.com/aarondl/null/v8"

	repo "knowledge-srv/internal/indexing/repository"
	"knowledge-srv/internal/sqlboiler"
)

// buildCreateDocument - Build sqlboiler entity from CreateDocumentOptions
func buildCreateDocument(opt repo.CreateDocumentOptions) *sqlboiler.IndexedDocument {
	dbDoc := &sqlboiler.IndexedDocument{
		AnalyticsID:    opt.AnalyticsID,
		ProjectID:      opt.ProjectID,
		SourceID:       opt.SourceID,
		QdrantPointID:  opt.QdrantPointID,
		CollectionName: opt.CollectionName,
		ContentHash:    opt.ContentHash,
		Status:         opt.Status,
		RetryCount:     null.IntFrom(opt.RetryCount),
		CreatedAt:      null.TimeFrom(time.Now()),
		UpdatedAt:      null.TimeFrom(time.Now()),
	}

	// Handle nullable fields
	if opt.ErrorMessage != nil {
		dbDoc.ErrorMessage = null.StringFrom(*opt.ErrorMessage)
	}
	if opt.BatchID != nil {
		dbDoc.BatchID = null.StringFrom(*opt.BatchID)
	}
	if opt.EmbeddingTimeMs > 0 {
		dbDoc.EmbeddingTimeMS = null.IntFrom(opt.EmbeddingTimeMs)
	}
	if opt.UpsertTimeMs > 0 {
		dbDoc.UpsertTimeMS = null.IntFrom(opt.UpsertTimeMs)
	}
	if opt.TotalTimeMs > 0 {
		dbDoc.TotalTimeMS = null.IntFrom(opt.TotalTimeMs)
	}
	if opt.IndexedAt != nil {
		dbDoc.IndexedAt = null.TimeFrom(*opt.IndexedAt)
	}

	return dbDoc
}

// buildUpsertDocument - Build sqlboiler entity from UpsertDocumentOptions
func buildUpsertDocument(opt repo.UpsertDocumentOptions) *sqlboiler.IndexedDocument {
	dbDoc := &sqlboiler.IndexedDocument{
		AnalyticsID:    opt.AnalyticsID,
		ProjectID:      opt.ProjectID,
		SourceID:       opt.SourceID,
		QdrantPointID:  opt.QdrantPointID,
		CollectionName: opt.CollectionName,
		ContentHash:    opt.ContentHash,
		Status:         opt.Status,
		RetryCount:     null.IntFrom(opt.RetryCount),
		UpdatedAt:      null.TimeFrom(time.Now()),
	}

	// Handle nullable fields
	if opt.ErrorMessage != nil {
		dbDoc.ErrorMessage = null.StringFrom(*opt.ErrorMessage)
	}
	if opt.BatchID != nil {
		dbDoc.BatchID = null.StringFrom(*opt.BatchID)
	}
	if opt.EmbeddingTimeMs > 0 {
		dbDoc.EmbeddingTimeMS = null.IntFrom(opt.EmbeddingTimeMs)
	}
	if opt.UpsertTimeMs > 0 {
		dbDoc.UpsertTimeMS = null.IntFrom(opt.UpsertTimeMs)
	}
	if opt.TotalTimeMs > 0 {
		dbDoc.TotalTimeMS = null.IntFrom(opt.TotalTimeMs)
	}
	if opt.IndexedAt != nil {
		dbDoc.IndexedAt = null.TimeFrom(*opt.IndexedAt)
	}

	return dbDoc
}
