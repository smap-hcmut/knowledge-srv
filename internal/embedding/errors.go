package embedding

import "errors"

var (
	ErrEmptyText           = errors.New("embedding: empty text")
	ErrNoVectorReturned    = errors.New("embedding: no vector returned")
	ErrEmptyTexts          = errors.New("embedding: empty texts")
	ErrMismatchVectorCount = errors.New("embedding: mismatch vector count")
)
