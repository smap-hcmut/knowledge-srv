package redis

import (
	"knowledge-srv/internal/search/repository"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/redis"
)

type implCacheRepository struct {
	redis redis.IRedis
	l     log.Logger
}

// New - Factory
func New(redis redis.IRedis, l log.Logger) repository.CacheRepository {
	return &implCacheRepository{
		redis: redis,
		l:     l,
	}
}
