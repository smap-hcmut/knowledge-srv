package indexing

import "errors"

var (
	ErrEmbeddingFailed     = errors.New("indexing: embedding generation failed")
	ErrQdrantUpsertFailed  = errors.New("indexing: qdrant upsert failed")
	ErrCountDocument       = errors.New("indexing: count document failed")
	ErrSkippedByGate       = errors.New("indexing: skipped by should_index gate")
	ErrInsightTitleEmpty   = errors.New("indexing: insight title is empty")
	ErrDigestBuildFailed   = errors.New("indexing: digest prose build failed")
	ErrInsightSummaryEmpty = errors.New("indexing: insight summary is empty")
)
