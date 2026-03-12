package redis

import (
	"knowledge-srv/internal/embedding/repository"

	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/redis"
)

type implRepository struct {
	redis redis.IRedis
	l     log.Logger
}

func New(redis redis.IRedis, l log.Logger) repository.Repository {
	return &implRepository{
		redis: redis,
		l:     l,
	}
}
