package redis

import (
	"knowledge-srv/internal/search/repository"
	"knowledge-srv/pkg/log"
	pkgRedis "knowledge-srv/pkg/redis"
)

type implCacheRepository struct {
	redis pkgRedis.IRedis
	l     log.Logger
}

// New - Factory
func New(redis pkgRedis.IRedis, l log.Logger) repository.CacheRepository {
	return &implCacheRepository{
		redis: redis,
		l:     l,
	}
}
