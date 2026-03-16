package http

import (
	"errors"
	"knowledge-srv/internal/indexing"
	"net/http"

	pkgErrors "github.com/smap-hcmut/shared-libs/go/errors"
)

var (
	errFileNotFound       = &pkgErrors.HTTPError{Code: 1, Message: "File not found in MinIO", StatusCode: http.StatusNotFound}
	errFileDownloadFailed = &pkgErrors.HTTPError{Code: 2, Message: "Failed to download file from MinIO", StatusCode: http.StatusInternalServerError}
	errFileParseFailed    = &pkgErrors.HTTPError{Code: 3, Message: "Failed to parse JSONL file", StatusCode: http.StatusUnprocessableEntity}
	errEmbeddingFailed    = &pkgErrors.HTTPError{Code: 4, Message: "Failed to generate embedding", StatusCode: http.StatusInternalServerError}
	errQdrantFailed       = &pkgErrors.HTTPError{Code: 5, Message: "Failed to upsert to Qdrant", StatusCode: http.StatusInternalServerError}
	ErrMissingProjectID   = &pkgErrors.HTTPError{Code: 6, Message: "Missing project_id parameter", StatusCode: http.StatusBadRequest}
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
