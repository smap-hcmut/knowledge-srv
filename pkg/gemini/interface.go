package gemini

import (
	"context"
	"time"

	pkghttp "knowledge-srv/pkg/http"
)

// IGemini defines the interface for Google Gemini text generation.
// Implementations are safe for concurrent use.
type IGemini interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// NewGemini creates a new Gemini client. Model defaults to DefaultModel if empty.
// APIKey must be set; Generate will return an error if it is empty.
func NewGemini(cfg GeminiConfig) IGemini {
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	return &geminiImpl{
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		httpClient: pkghttp.NewClient(pkghttp.ClientConfig{
			Timeout:   60 * time.Second,
			Retries:   3,
			RetryWait: 1 * time.Second,
		}),
	}
}
