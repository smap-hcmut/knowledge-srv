package indexing

import "errors"

var (
	ErrAlreadyIndexed       = errors.New("indexing: record already indexed")
	ErrContentTooShort      = errors.New("indexing: content too short")
	ErrDuplicateContent     = errors.New("indexing: duplicate content")
	ErrEmbeddingFailed      = errors.New("indexing: embedding generation failed")
	ErrFileDownloadFailed   = errors.New("indexing: file download failed")
	ErrFileNotFound         = errors.New("indexing: file not found")
	ErrFileParseFailed      = errors.New("indexing: file parse failed")
	ErrInvalidAnalyticsData = errors.New("indexing: invalid analytics data")
	ErrQdrantUpsertFailed   = errors.New("indexing: qdrant upsert failed")
	ErrCountDocument        = errors.New("indexing: count document failed")
)
