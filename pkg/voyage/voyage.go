package voyage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

const (
	// voyageMaxAttempts caps the number of in-process retries for a 429 / 5xx
	// before the caller must take over (typically by parking the message
	// back on the Kafka topic for the next consumer pass).
	voyageMaxAttempts = 4

	// voyageBaseBackoff is the floor for exponential backoff. Jitter is
	// added on top so concurrent consumers do not synchronize their retries.
	voyageBaseBackoff = 500 * time.Millisecond

	// voyageMaxBackoff caps how long we'll sleep between retries so a single
	// embed call cannot hold a Kafka message hostage forever.
	voyageMaxBackoff = 8 * time.Second

	// voyageMaxBatchSize is Voyage API's max inputs per call. Larger batches
	// are split transparently.
	voyageMaxBatchSize = 128
)

// errVoyageRetryable signals the calling Embed loop that it can try again.
var errVoyageRetryable = errors.New("voyage: retryable upstream error")

// Embed generates embeddings for the given texts. Wraps the HTTP call in an
// exponential-backoff retry loop so transient 429 (rate limit) / 5xx
// responses do not fail entire batches — the previous behaviour dropped
// every doc in the batch on a single throttled response.
func (v *voyageImpl) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if v.apiKey == "" {
		return nil, fmt.Errorf("voyage: API key is required")
	}
	if len(texts) == 0 {
		return nil, fmt.Errorf("voyage: at least one text is required")
	}

	// Chunk into voyageMaxBatchSize to respect Voyage's 128 input limit.
	allEmbeddings := make([][]float32, 0, len(texts))
	for start := 0; start < len(texts); start += voyageMaxBatchSize {
		end := start + voyageMaxBatchSize
		if end > len(texts) {
			end = len(texts)
		}
		chunk := texts[start:end]
		chunkEmb, err := v.embedChunk(ctx, chunk)
		if err != nil {
			return nil, err
		}
		allEmbeddings = append(allEmbeddings, chunkEmb...)
	}
	return allEmbeddings, nil
}

// embedChunk sends one batch (≤ voyageMaxBatchSize) to Voyage with retries.
func (v *voyageImpl) embedChunk(ctx context.Context, texts []string) ([][]float32, error) {
	req := Request{
		Input: texts,
		Model: Model,
	}
	headers := map[string]string{"Authorization": "Bearer " + v.apiKey}

	var lastErr error
	for attempt := 1; attempt <= voyageMaxAttempts; attempt++ {
		embeddings, err := v.embedOnce(ctx, req, headers)
		if err == nil {
			return embeddings, nil
		}
		lastErr = err
		if !errors.Is(err, errVoyageRetryable) || attempt == voyageMaxAttempts {
			return nil, err
		}
		// Capped exponential backoff with jitter; respects ctx cancellation
		// so a shutting-down consumer does not block on a sleep.
		sleep := voyageBaseBackoff << (attempt - 1)
		if sleep > voyageMaxBackoff {
			sleep = voyageMaxBackoff
		}
		sleep += time.Duration(rand.Int63n(int64(voyageBaseBackoff)))
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(sleep):
		}
	}

	return nil, lastErr
}

// embedOnce performs a single HTTP call. Returns errVoyageRetryable for
// transient upstream failures so the outer loop can back off.
func (v *voyageImpl) embedOnce(ctx context.Context, req Request, headers map[string]string) ([][]float32, error) {
	body, statusCode, err := v.httpClient.Post(ctx, Endpoint, req, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to call voyage API: %w", err)
	}

	if statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError {
		return nil, fmt.Errorf("%w: voyage API returned status %d, body: %s", errVoyageRetryable, statusCode, string(body))
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("voyage API returned status: %d, body: %s", statusCode, string(body))
	}

	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal voyage response: %w", err)
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, item := range resp.Data {
		embeddings[i] = item.Embedding
	}

	return embeddings, nil
}
