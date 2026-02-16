package redis

import (
	"knowledge-srv/internal/embedding/repository"
	"knowledge-srv/pkg/log"
	pkgRedis "knowledge-srv/pkg/redis"
)

type implRepository struct {
	redis pkgRedis.IRedis
	l     log.Logger
}

func New(redis pkgRedis.IRedis, l log.Logger) repository.Repository {
	return &implRepository{
		redis: redis,
		l:     l,
	}
}
