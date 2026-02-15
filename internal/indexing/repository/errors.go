package repository

import "errors"

var (
	ErrNotFound      = errors.New("project not found")
	ErrAlreadyExists = errors.New("project already exists")
	ErrInvalidInput  = errors.New("invalid input")
)
