package search

import "errors"

var (
	ErrCampaignNotFound   = errors.New("search: campaign not found")
	ErrCampaignNoProjects = errors.New("search: campaign has no projects")
	ErrQueryTooShort      = errors.New("search: query too short")
	ErrQueryTooLong       = errors.New("search: query too long")
	ErrEmbeddingFailed    = errors.New("search: embedding generation failed")
	ErrSearchFailed       = errors.New("search: qdrant search failed")
	ErrInvalidFilters     = errors.New("search: invalid filters")
)
