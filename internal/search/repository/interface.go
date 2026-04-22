package repository

import (
	"context"
)

// QdrantRepository is removed in favor of point.UseCase

//go:generate mockery --name CacheRepository
type CacheRepository interface {
	GetCampaignProjects(ctx context.Context, campaignID string) ([]string, error)
	SaveCampaignProjects(ctx context.Context, campaignID string, projectIDs []string) error

	GetCampaignName(ctx context.Context, campaignID string) (string, error)
	SaveCampaignName(ctx context.Context, campaignID string, name string) error

	GetSearchResults(ctx context.Context, cacheKey string) ([]byte, error)
	SaveSearchResults(ctx context.Context, cacheKey string, data []byte) error

	InvalidateSearchCache(ctx context.Context, projectID string) error
}
