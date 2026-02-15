package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	pkghttp "knowledge-srv/pkg/http"
)

// Gemini interacts with Google Gemini API
type Gemini struct {
	apiKey     string
	model      string
	httpClient *pkghttp.Client
}

// NewGemini creates a new Gemini client
func NewGemini(apiKey, model string) *Gemini {
	if model == "" {
		model = DefaultModel
	}
	return &Gemini{
		apiKey: apiKey,
		model:  model,
		httpClient: pkghttp.NewClient(pkghttp.ClientConfig{
			Timeout:   60 * time.Second,
			Retries:   3,
			RetryWait: 1 * time.Second,
		}),
	}
}

// Generate generates content based on the prompt
func (g *Gemini) Generate(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", BaseURL, g.model, g.apiKey)

	req := Request{
		Contents: []Content{
			{
				Parts: []Part{
					{Text: prompt},
				},
			},
		},
	}

	body, statusCode, err := g.httpClient.Post(ctx, url, req, nil)
	if err != nil {
		return "", fmt.Errorf("failed to call Gemini API: %w", err)
	}

	if statusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini API returned status: %d, body: %s", statusCode, string(body))
	}

	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal Gemini response: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	var generatedText string
	for _, part := range resp.Candidates[0].Content.Parts {
		generatedText += part.Text
	}

	return generatedText, nil
}
