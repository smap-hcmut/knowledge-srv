package repository

import "errors"

var (
	// ErrNotFound is returned when a row does not exist.
	ErrNotFound = errors.New("notebook repository: not found")
	// ErrChatJobNotFound is returned when a chat job id is unknown.
	ErrChatJobNotFound = errors.New("notebook repository: chat job not found")
)
