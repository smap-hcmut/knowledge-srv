package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	pkghttp "knowledge-srv/pkg/http"
)

// VoyageClient interacts with Voyage AI API
type VoyageClient struct {
	apiKey     string
	httpClient *pkghttp.Client
}

// NewVoyageClient creates a new Voyage AI client
func NewVoyageClient(apiKey string) *VoyageClient {
	return &VoyageClient{
		apiKey: apiKey,
		httpClient: pkghttp.NewClient(pkghttp.ClientConfig{
			Timeout:   30 * time.Second,
			Retries:   3,
			RetryWait: 1 * time.Second,
		}),
	}
}

// Embed generates embeddings for the given texts
func (c *VoyageClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	url := "https://api.voyageai.com/v1/embeddings"

	payload := map[string]interface{}{
		"input": texts,
		"model": "voyage-multilingual-2",
	}

	headers := map[string]string{
		"Authorization": "Bearer " + c.apiKey,
	}

	body, statusCode, err := c.httpClient.Post(ctx, url, payload, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to call Voyage API: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("Voyage API returned status: %d, body: %s", statusCode, string(body))
	}

	var response struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Voyage response: %w", err)
	}

	embeddings := make([][]float32, len(response.Data))
	for i, item := range response.Data {
		embeddings[i] = item.Embedding
	}

	return embeddings, nil
}
