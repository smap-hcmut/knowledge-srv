package voyage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	pkghttp "knowledge-srv/pkg/http"
)

// Voyage interacts with Voyage AI API
type Voyage struct {
	apiKey     string
	httpClient *pkghttp.Client
}

// NewVoyage creates a new Voyage AI client
func NewVoyage(apiKey string) *Voyage {
	return &Voyage{
		apiKey: apiKey,
		httpClient: pkghttp.NewClient(pkghttp.ClientConfig{
			Timeout:   30 * time.Second,
			Retries:   3,
			RetryWait: 1 * time.Second,
		}),
	}
}

// Embed generates embeddings for the given texts
func (v *Voyage) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	req := Request{
		Input: texts,
		Model: Model,
	}

	headers := map[string]string{
		"Authorization": "Bearer " + v.apiKey,
	}

	body, statusCode, err := v.httpClient.Post(ctx, Endpoint, req, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to call Voyage API: %w", err)
	}

	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("Voyage API returned status: %d, body: %s", statusCode, string(body))
	}

	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Voyage response: %w", err)
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, item := range resp.Data {
		embeddings[i] = item.Embedding
	}

	return embeddings, nil
}
