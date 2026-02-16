package redis

import (
	"context"
	"crypto/sha256"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

// InvalidateSearchCache removes all search-related cache keys for a project
// using Redis SCAN + pipelined DELETE.
func (r *implCacheRepository) InvalidateSearchCache(ctx context.Context, projectID string) error {
	pattern := fmt.Sprintf("search:*%s*", projectID)
	client := r.redis.GetClient()

	var cursor uint64
	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			r.l.Errorf(ctx, "indexing.repository.redis.InvalidateSearchCache: Failed to scan cache: %v", err)
			return err
		}

		if len(keys) > 0 {
			pipe := client.Pipeline()
			for _, key := range keys {
				pipe.Del(ctx, key)
			}
			if _, err := pipe.Exec(ctx); err != nil && err != goredis.Nil {
				r.l.Errorf(ctx, "indexing.repository.redis.InvalidateSearchCache: Failed to execute pipeline: %v", err)
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

// embeddingCacheKey generates a Redis key from content hash.
func embeddingCacheKey(contentHash string) string {
	return fmt.Sprintf("embedding:%s", contentHash)
}

// ContentHash generates a SHA-256 hash of content for use as cache key.
func ContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}
