package repository

import pb "github.com/qdrant/go-client/qdrant"

// SearchPointsOptions - Options cho SearchPoints
type SearchPointsOptions struct {
	Vector         []float32  // Query vector
	Limit          uint64     // Max results
	Filter         *pb.Filter // Qdrant payload filter
	ScoreThreshold *float32   // Min score threshold (optional)
	WithPayload    []string   // Fields to include in payload (selective retrieval)
}
