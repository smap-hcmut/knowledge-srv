package search

import "errors"

// Domain errors
var (
	// ErrCampaignNotFound - Campaign không tồn tại hoặc user không có quyền
	ErrCampaignNotFound = errors.New("search: campaign not found")

	// ErrCampaignNoProjects - Campaign không có project nào
	ErrCampaignNoProjects = errors.New("search: campaign has no projects")

	// ErrQueryTooShort - Query text quá ngắn (< 3 chars)
	ErrQueryTooShort = errors.New("search: query too short")

	// ErrQueryTooLong - Query text quá dài (> 1000 chars)
	ErrQueryTooLong = errors.New("search: query too long")

	// ErrEmbeddingFailed - Không sinh được embedding cho query
	ErrEmbeddingFailed = errors.New("search: embedding generation failed")

	// ErrSearchFailed - Qdrant search thất bại
	ErrSearchFailed = errors.New("search: qdrant search failed")

	// ErrInvalidFilters - Filters không hợp lệ
	ErrInvalidFilters = errors.New("search: invalid filters")
)
