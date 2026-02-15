package http

import (
	"errors"
	"net/http"

	"knowledge-srv/internal/indexing"
	pkgErrors "knowledge-srv/pkg/errors"
)

// Error codes
var (
	errFileNotFound = pkgErrors.NewHTTPError(
		http.StatusBadRequest,
		"File not found in MinIO",
	)

	errFileDownloadFailed = pkgErrors.NewHTTPError(
		http.StatusInternalServerError,
		"Failed to download file from MinIO",
	)

	errFileParseFailed = pkgErrors.NewHTTPError(
		http.StatusBadRequest,
		"Failed to parse JSONL file",
	)

	errEmbeddingFailed = pkgErrors.NewHTTPError(
		http.StatusInternalServerError,
		"Failed to generate embedding",
	)

	errQdrantFailed = pkgErrors.NewHTTPError(
		http.StatusInternalServerError,
		"Failed to upsert to Qdrant",
	)
)

// mapError - Map domain errors to HTTP errors
func (h *handler) mapError(err error) error {
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
		// Unknown error → panic (theo convention)
		panic(err)
	}
}
