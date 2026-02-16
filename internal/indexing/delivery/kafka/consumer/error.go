package consumer

import (
	"errors"
)

var (
	ErrConsumerGroupNotFound     = errors.New("consumer group not found")
	ErrCreateConsumerGroupFailed = errors.New("failed to create consumer group")
)
