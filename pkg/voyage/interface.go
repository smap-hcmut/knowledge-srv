package voyage

import "context"

// IVoyage defines the interface for Voyage AI interactions
type IVoyage interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}
