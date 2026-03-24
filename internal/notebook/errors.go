package notebook

import "errors"

var (
	ErrSessionNotFound   = errors.New("notebook: session not found")
	ErrNotebookNotFound  = errors.New("notebook: notebook not found")
	ErrNotebookExists    = errors.New("notebook: notebook already exists")
	ErrSyncFailed        = errors.New("notebook: sync operation failed")
	ErrWebhookValidation = errors.New("notebook: webhook validation failed")
	ErrInvalidInput      = errors.New("notebook: invalid input provided")
	ErrJobTimeout        = errors.New("notebook: job processing timeout")
	ErrJobFailed         = errors.New("notebook: job failed")
)
