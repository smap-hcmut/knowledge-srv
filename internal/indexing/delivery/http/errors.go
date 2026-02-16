package http

import (
	"errors"

	"knowledge-srv/internal/indexing"
	pkgErrors "knowledge-srv/pkg/errors"
)

var (
	errFileNotFound       = pkgErrors.NewHTTPError(30001, "File not found in MinIO")
	errFileDownloadFailed = pkgErrors.NewHTTPError(30002, "Failed to download file from MinIO")
	errFileParseFailed    = pkgErrors.NewHTTPError(30003, "Failed to parse JSONL file")
	errEmbeddingFailed    = pkgErrors.NewHTTPError(30004, "Failed to generate embedding")
	errQdrantFailed       = pkgErrors.NewHTTPError(30005, "Failed to upsert to Qdrant")
	ErrMissingProjectID   = pkgErrors.NewHTTPError(30006, "Missing project_id parameter")
)

var NotFound = []error{
	errFileNotFound,
}

func (h handler) mapError(err error) error {
	switch {
	case errors.Is(err, indexing.ErrFileNotFound):
		return errFileNotFound
	case errors.Is(err, indexing.ErrFileDownloadFailed):
		return errFileDownloadFailed
	case errors.Is(err, indexing.ErrFileParseFailed):
		return errFileParseFailed
	case errors.Is(err, indexing.ErrEmbeddingFailed):
		return errEmbeddingFailed
	case errors.Is(err, indexing.ErrQdrantUpsertFailed):
		return errQdrantFailed
	default:
		return err
	}
}
