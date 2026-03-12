package redis

import (
	repo "knowledge-srv/internal/indexing/repository"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/redis"
)

type implCacheRepository struct {
	redis redis.IRedis
	l     log.Logger
}

// New creates a new CacheRepository backed by Redis.
func New(redis redis.IRedis, l log.Logger) repo.CacheRepository {
	return &implCacheRepository{
		redis: redis,
		l:     l,
	}
}
