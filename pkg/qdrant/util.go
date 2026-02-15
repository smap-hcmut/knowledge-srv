package qdrant

import (
	"fmt"

	pb "github.com/qdrant/go-client/qdrant"
)

// Validate validates the Qdrant configuration
func (cfg Config) Validate() error {
	if cfg.Host == "" {
		return fmt.Errorf("%w: host is required", ErrInvalidConfig)
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("%w: invalid port number", ErrInvalidConfig)
	}
	return nil
}

// GetDefaultConfig returns a default Qdrant configuration
func GetDefaultConfig() Config {
	return Config{
		Host:   "localhost",
		Port:   6334,
		UseTLS: false,
	}
}

// GetDistanceMetric returns the appropriate distance metric
func GetDistanceMetric(metric string) pb.Distance {
	switch metric {
	case "cosine":
		return pb.Distance_Cosine
	case "euclidean":
		return pb.Distance_Euclid
	case "dot":
		return pb.Distance_Dot
	case "manhattan":
		return pb.Distance_Manhattan
	default:
		return pb.Distance_Cosine
	}
}

// ValidateVector checks if a vector is valid
func ValidateVector(vector []float32, expectedSize uint64) error {
	if len(vector) == 0 {
		return ErrInvalidVector
	}
	if uint64(len(vector)) != expectedSize {
		return fmt.Errorf("%w: expected size %d, got %d", ErrInvalidVector, expectedSize, len(vector))
	}
	return nil
}

// ValidateVectors validates multiple vectors
func ValidateVectors(vectors [][]float32, expectedSize uint64) error {
	if len(vectors) == 0 {
		return ErrInvalidVector
	}
	for i, vector := range vectors {
		if err := ValidateVector(vector, expectedSize); err != nil {
			return fmt.Errorf("invalid vector at index %d: %w", i, err)
		}
	}
	return nil
}
