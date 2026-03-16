package voyage

import (
	"context"
	"fmt"

	pkghttp "github.com/smap-hcmut/shared-libs/go/httpclient"
)

// IVoyage defines the interface for Voyage AI embeddings.
// Implementations are safe for concurrent use.
type IVoyage interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// NewVoyage creates a new Voyage client. APIKey must be set; Embed returns an error if it is empty.
func NewVoyage(cfg VoyageConfig) (IVoyage, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("voyage: API key is required")
	}
	return &voyageImpl{
		apiKey: cfg.APIKey,
		httpClient: pkghttp.NewDefaultClient(),
	}, nil
}
