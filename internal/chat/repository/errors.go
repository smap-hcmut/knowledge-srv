package repository

import "errors"

var (
	ErrFailedToInsert = errors.New("failed to insert")
	ErrFailedToGet    = errors.New("failed to get")
	ErrFailedToList   = errors.New("failed to list")
	ErrFailedToUpdate = errors.New("failed to update")
)
