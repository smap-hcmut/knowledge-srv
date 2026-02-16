package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// =====================================================
// Tầng 2: Campaign Projects Cache (TTL 10 min)
// =====================================================

func (r *implCacheRepository) GetCampaignProjects(ctx context.Context, campaignID string) ([]string, error) {
	key := fmt.Sprintf("campaign_projects:%s", campaignID)
	data, err := r.redis.GetClient().Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var projectIDs []string
	if err := json.Unmarshal([]byte(data), &projectIDs); err != nil {
		r.l.Errorf(ctx, "search.repository.redis.GetCampaignProjects: Failed to unmarshal project IDs: %v", err)
		return nil, err
	}
	return projectIDs, nil
}

func (r *implCacheRepository) SaveCampaignProjects(ctx context.Context, campaignID string, projectIDs []string) error {
	key := fmt.Sprintf("campaign_projects:%s", campaignID)
	data, err := json.Marshal(projectIDs)
	if err != nil {
		return err
	}
	if err := r.redis.GetClient().Set(ctx, key, data, 10*time.Minute).Err(); err != nil {
		r.l.Errorf(ctx, "search.repository.redis.SaveCampaignProjects: Failed to save to cache: %v", err)
		return err
	}
	return nil
}

// =====================================================
// Tầng 3: Search Results Cache (TTL 5 min)
// =====================================================

func (r *implCacheRepository) GetSearchResults(ctx context.Context, cacheKey string) ([]byte, error) {
	data, err := r.redis.GetClient().Get(ctx, cacheKey).Result()
	if err != nil {
		return nil, err
	}
	return []byte(data), nil
}

func (r *implCacheRepository) SaveSearchResults(ctx context.Context, cacheKey string, data []byte) error {
	if err := r.redis.GetClient().Set(ctx, cacheKey, data, 5*time.Minute).Err(); err != nil {
		r.l.Errorf(ctx, "search.repository.redis.SaveSearchResults: Failed to save to cache: %v", err)
		return err
	}
	return nil
}

// =====================================================
// Cache Invalidation
// =====================================================

func (r *implCacheRepository) InvalidateSearchCache(ctx context.Context, projectID string) error {
	pattern := fmt.Sprintf("search:*%s*", projectID)
	client := r.redis.GetClient()

	var cursor uint64
	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			r.l.Errorf(ctx, "search.repository.redis.InvalidateSearchCache: Failed to scan cache: %v", err)
			return err
		}
		if len(keys) > 0 {
			pipe := client.Pipeline()
			for _, key := range keys {
				pipe.Del(ctx, key)
			}
			if _, err := pipe.Exec(ctx); err != nil && err != goredis.Nil {
				r.l.Errorf(ctx, "search.repository.redis.InvalidateSearchCache: Failed to execute pipeline: %v", err)
				return err
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}
