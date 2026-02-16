package http

import (
	"errors"
	"knowledge-srv/internal/search"
	pkgErrors "knowledge-srv/pkg/errors"
)

var (
	errCampaignNotFound = pkgErrors.NewHTTPError(
		404, "Campaign not found",
	)
	errCampaignNoProjects = pkgErrors.NewHTTPError(
		400, "Campaign has no projects",
	)
	errQueryTooShort = pkgErrors.NewHTTPError(
		400, "Query too short (min 3 characters)",
	)
	errQueryTooLong = pkgErrors.NewHTTPError(
		400, "Query too long (max 1000 characters)",
	)
	errEmbeddingFailed = pkgErrors.NewHTTPError(
		500, "Failed to generate query embedding",
	)
	errSearchFailed = pkgErrors.NewHTTPError(
		500, "Search failed",
	)
	errInvalidFilters = pkgErrors.NewHTTPError(
		400, "Invalid search filters",
	)
)

func (h *handler) mapError(err error) error {
	switch {
	case errors.Is(err, search.ErrCampaignNotFound):
		return errCampaignNotFound
	case errors.Is(err, search.ErrCampaignNoProjects):
		return errCampaignNoProjects
	case errors.Is(err, search.ErrQueryTooShort):
		return errQueryTooShort
	case errors.Is(err, search.ErrQueryTooLong):
		return errQueryTooLong
	case errors.Is(err, search.ErrEmbeddingFailed):
		return errEmbeddingFailed
	case errors.Is(err, search.ErrSearchFailed):
		return errSearchFailed
	case errors.Is(err, search.ErrInvalidFilters):
		return errInvalidFilters
	default:
		panic(err)
	}
}
