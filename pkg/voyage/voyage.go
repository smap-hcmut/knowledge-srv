package voyage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Embed generates embeddings for the given texts.
func (v *voyageImpl) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if v.apiKey == "" {
		return nil, fmt.Errorf("voyage: API key is required")
	}
	if len(texts) == 0 {
		return nil, fmt.Errorf("voyage: at least one text is required")
	}

	req := Request{
		Input: texts,
		Model: Model,
	}

	headers := map[string]string{"Authorization": "Bearer " + v.apiKey}

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
