package gemini

import (
	"context"
	"fmt"
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
func NewGemini(cfg GeminiConfig) (IGemini, error) {
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("gemini: API key is required")
	}
	return &geminiImpl{
		apiKey: cfg.APIKey,
		model:  cfg.Model,
		httpClient: pkghttp.NewClient(pkghttp.ClientConfig{
			Timeout:   60 * time.Second,
			Retries:   3,
			RetryWait: 1 * time.Second,
		}),
	}, nil
}
