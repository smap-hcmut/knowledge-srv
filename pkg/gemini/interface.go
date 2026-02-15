package gemini

import "context"

// IGemini defines the interface for Google Gemini interactions
type IGemini interface {
	Generate(ctx context.Context, prompt string) (string, error)
}
