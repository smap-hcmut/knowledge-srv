package http

import (
	"knowledge-srv/internal/indexing"
	pkgErrors "knowledge-srv/pkg/errors"
)

var (
	errFileNotFound       = pkgErrors.NewHTTPError(30001, "File not found in MinIO")
	errFileDownloadFailed = pkgErrors.NewHTTPError(30002, "Failed to download file from MinIO")
	errFileParseFailed    = pkgErrors.NewHTTPError(30003, "Failed to parse JSONL file")
	errEmbeddingFailed    = pkgErrors.NewHTTPError(30004, "Failed to generate embedding")
	errQdrantFailed       = pkgErrors.NewHTTPError(30005, "Failed to upsert to Qdrant")
)

var NotFound = []error{
	errFileNotFound,
}

func (h handler) mapError(err error) error {
	switch err {
	case indexing.ErrFileNotFound:
		return errFileNotFound
	case indexing.ErrFileDownloadFailed:
		return errFileDownloadFailed
	case indexing.ErrFileParseFailed:
		return errFileParseFailed
	case indexing.ErrEmbeddingFailed:
		return errEmbeddingFailed
	case indexing.ErrQdrantUpsertFailed:
		return errQdrantFailed
	default:
		return err
	}
}
