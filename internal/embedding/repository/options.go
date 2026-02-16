package repository

import "time"

type GetOptions struct {
	Key string
}

type SaveOptions struct {
	Key    string
	Vector []float32
	TTL    time.Duration
}
