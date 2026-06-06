package http

import (
	"errors"
	"knowledge-srv/internal/indexing"
	"net/http"

	pkgErrors "github.com/smap-hcmut/shared-libs/go/errors"
)

var (
	errEmbeddingFailed  = &pkgErrors.HTTPError{Code: 4, Message: "Failed to generate embedding", StatusCode: http.StatusInternalServerError}
	errQdrantFailed     = &pkgErrors.HTTPError{Code: 5, Message: "Failed to upsert to Qdrant", StatusCode: http.StatusInternalServerError}
	ErrMissingProjectID = &pkgErrors.HTTPError{Code: 6, Message: "Missing project_id parameter", StatusCode: http.StatusBadRequest}
)

func (h handler) mapError(err error) error {
	switch {
	case errors.Is(err, indexing.ErrEmbeddingFailed):
		return errEmbeddingFailed
	case errors.Is(err, indexing.ErrQdrantUpsertFailed):
		return errQdrantFailed
	default:
		return err
	}
}
