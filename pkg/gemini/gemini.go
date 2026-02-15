package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Generate generates content based on the prompt.
func (g *geminiImpl) Generate(ctx context.Context, prompt string) (string, error) {
	if g.apiKey == "" {
		return "", fmt.Errorf("gemini: API key is required")
	}
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

	var b strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		b.WriteString(part.Text)
	}
	return b.String(), nil
}
