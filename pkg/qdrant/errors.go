package qdrant

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidConfig      = errors.New("invalid configuration")
	ErrCollectionNotFound = errors.New("collection not found")
	ErrPointNotFound      = errors.New("point not found")
	ErrInvalidVector      = errors.New("invalid vector")
	ErrInvalidPointID     = errors.New("invalid point ID")
	ErrEmptyCollection    = errors.New("collection name cannot be empty")
	ErrInvalidVectorSize  = errors.New("invalid vector size")
	ErrConnectionFailed   = errors.New("connection failed")
	ErrEmptyKey           = errors.New("facet key cannot be empty")
	ErrMissingGroupField  = errors.New("groupBy field is required")
)

// WrapError wraps an error with additional context.
func WrapError(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}
