package voyage

import (
	"context"
	"time"

	pkghttp "knowledge-srv/pkg/http"
)

// IVoyage defines the interface for Voyage AI embeddings.
// Implementations are safe for concurrent use.
type IVoyage interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// NewVoyage creates a new Voyage client. APIKey must be set; Embed returns an error if it is empty.
func NewVoyage(cfg VoyageConfig) IVoyage {
	return &voyageImpl{
		apiKey: cfg.APIKey,
		httpClient: pkghttp.NewClient(pkghttp.ClientConfig{
			Timeout:   30 * time.Second,
			Retries:  3,
			RetryWait: 1 * time.Second,
		}),
	}
}
