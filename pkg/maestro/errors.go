package maestro

import (
	"errors"
	"fmt"
)

var (
	ErrSessionNotFound  = errors.New("maestro: session not found")
	ErrSessionBusy      = errors.New("maestro: session is busy")
	ErrJobNotFound      = errors.New("maestro: job not found")
	ErrJobFailed        = errors.New("maestro: job failed")
	ErrUnauthorized     = errors.New("maestro: unauthorized (invalid API key)")
	ErrRateLimited      = errors.New("maestro: rate limited")
	ErrNotebookNotFound = errors.New("maestro: notebook not found")
	ErrBadRequest       = errors.New("maestro: bad request")
	ErrServerError      = errors.New("maestro: server error")
)

// WrapError wraps an error with additional context.
func WrapError(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}
