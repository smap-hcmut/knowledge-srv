package repository

import "errors"

var (
	ErrFailedToSearch    = errors.New("repository: failed to search points")
	ErrFailedToCount     = errors.New("repository: failed to count points")
	ErrCacheMiss         = errors.New("repository: cache miss")
	ErrCacheSetFailed    = errors.New("repository: failed to set cache")
	ErrCacheDeleteFailed = errors.New("repository: failed to delete cache")
)
