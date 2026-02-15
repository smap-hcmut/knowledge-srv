package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	pkghttp "knowledge-srv/pkg/http"
)

// GeminiClient interacts with Google Gemini API
type GeminiClient struct {
	apiKey     string
	model      string
	httpClient *pkghttp.Client
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(apiKey, model string) *GeminiClient {
	if model == "" {
		model = "gemini-1.5-pro"
	}
	return &GeminiClient{
		apiKey: apiKey,
		model:  model,
		httpClient: pkghttp.NewClient(pkghttp.ClientConfig{
			Timeout:   60 * time.Second, // Gemini can take longer
			Retries:   3,
			RetryWait: 1 * time.Second,
		}),
	}
}

// Generate generates content based on the prompt
func (c *GeminiClient) Generate(ctx context.Context, prompt string) (string, error) {
	// https://ai.google.dev/api/rest/v1/models/generateContent
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{
						"text": prompt,
					},
				},
			},
		},
	}

	body, statusCode, err := c.httpClient.Post(ctx, url, payload, nil)
	if err != nil {
		return "", fmt.Errorf("failed to call Gemini API: %w", err)
	}

	if statusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini API returned status: %d, body: %s", statusCode, string(body))
	}

	// Define response structure based on Google AI API
	var response struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal Gemini response: %w", err)
	}

	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	// Aggregate all parts
	var generatedText string
	for _, part := range response.Candidates[0].Content.Parts {
		generatedText += part.Text
	}

	return generatedText, nil
}
