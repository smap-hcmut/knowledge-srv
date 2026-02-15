package qdrant

import "time"

// Config holds Qdrant configuration
type Config struct {
	Host    string
	Port    int
	UseTLS  bool
	APIKey  string
	Timeout time.Duration
}

// Point represents a vector point in Qdrant
type Point struct {
	ID      string
	Vector  []float32
	Payload map[string]interface{}
}

// SearchResult represents a search result from Qdrant
type SearchResult struct {
	ID      string
	Score   float32
	Payload map[string]interface{}
}

// CollectionInfo represents collection metadata
type CollectionInfo struct {
	Name        string
	VectorSize  uint64
	Distance    string
	PointsCount uint64
	Status      string
}
