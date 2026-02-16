package repository

import "errors"

var (
	ErrNotFound             = errors.New("project not found")
	ErrAlreadyExists        = errors.New("project already exists")
	ErrInvalidInput         = errors.New("invalid input")
	ErrFailedToInsert       = errors.New("failed to insert")
	ErrFailedToGet          = errors.New("failed to get")
	ErrFailedToList         = errors.New("failed to list")
	ErrFailedToMarkResolved = errors.New("failed to mark resolved")
	ErrFailedToCount        = errors.New("failed to count")
	ErrFailedToUpdateStatus = errors.New("failed to update status")
	ErrFailedToUpsert       = errors.New("failed to upsert")
)
