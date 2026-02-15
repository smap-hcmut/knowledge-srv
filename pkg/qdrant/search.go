package qdrant

import (
	"context"

	pb "github.com/qdrant/go-client/qdrant"
)

// Search performs a vector similarity search
func (c *Client) Search(ctx context.Context, collectionName string, vector []float32, limit uint64) ([]SearchResult, error) {
	if collectionName == "" {
		return nil, ErrEmptyCollection
	}
	if len(vector) == 0 {
		return nil, ErrInvalidVector
	}
	if limit == 0 {
		limit = 10
	}

	resp, err := c.pointsClient.Search(ctx, &pb.SearchPoints{
		CollectionName: collectionName,
		Vector:         vector,
		Limit:          limit,
		WithPayload: &pb.WithPayloadSelector{
			SelectorOptions: &pb.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
	})

	if err != nil {
		return nil, WrapError(err, "failed to search")
	}

	results := make([]SearchResult, 0, len(resp.Result))
	for _, hit := range resp.Result {
		payload := make(map[string]interface{})
		for key, value := range hit.Payload {
			payload[key] = value.AsInterface()
		}

		var id string
		if hit.Id != nil {
			id = hit.Id.GetUuid()
		}

		results = append(results, SearchResult{
			ID:      id,
			Score:   hit.Score,
			Payload: payload,
		})
	}

	return results, nil
}

// SearchWithFilter performs a vector similarity search with payload filter
func (c *Client) SearchWithFilter(ctx context.Context, collectionName string, vector []float32, limit uint64, filter *pb.Filter) ([]SearchResult, error) {
	if collectionName == "" {
		return nil, ErrEmptyCollection
	}
	if len(vector) == 0 {
		return nil, ErrInvalidVector
	}
	if limit == 0 {
		limit = 10
	}

	resp, err := c.pointsClient.Search(ctx, &pb.SearchPoints{
		CollectionName: collectionName,
		Vector:         vector,
		Limit:          limit,
		Filter:         filter,
		WithPayload: &pb.WithPayloadSelector{
			SelectorOptions: &pb.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
	})

	if err != nil {
		return nil, WrapError(err, "failed to search with filter")
	}

	results := make([]SearchResult, 0, len(resp.Result))
	for _, hit := range resp.Result {
		payload := make(map[string]interface{})
		for key, value := range hit.Payload {
			payload[key] = value.AsInterface()
		}

		var id string
		if hit.Id != nil {
			id = hit.Id.GetUuid()
		}

		results = append(results, SearchResult{
			ID:      id,
			Score:   hit.Score,
			Payload: payload,
		})
	}

	return results, nil
}

// SearchBatch performs multiple vector searches in a single request
func (c *Client) SearchBatch(ctx context.Context, collectionName string, vectors [][]float32, limit uint64) ([][]SearchResult, error) {
	if collectionName == "" {
		return nil, ErrEmptyCollection
	}
	if len(vectors) == 0 {
		return nil, ErrInvalidVector
	}
	if limit == 0 {
		limit = 10
	}

	searches := make([]*pb.SearchPoints, 0, len(vectors))
	for _, vector := range vectors {
		if len(vector) == 0 {
			return nil, ErrInvalidVector
		}

		searches = append(searches, &pb.SearchPoints{
			CollectionName: collectionName,
			Vector:         vector,
			Limit:          limit,
			WithPayload: &pb.WithPayloadSelector{
				SelectorOptions: &pb.WithPayloadSelector_Enable{
					Enable: true,
				},
			},
		})
	}

	resp, err := c.pointsClient.SearchBatch(ctx, &pb.SearchBatchPoints{
		CollectionName: collectionName,
		SearchPoints:   searches,
	})

	if err != nil {
		return nil, WrapError(err, "failed to batch search")
	}

	allResults := make([][]SearchResult, 0, len(resp.Result))
	for _, batchResult := range resp.Result {
		results := make([]SearchResult, 0, len(batchResult.Result))
		for _, hit := range batchResult.Result {
			payload := make(map[string]interface{})
			for key, value := range hit.Payload {
				payload[key] = value.AsInterface()
			}

			var id string
			if hit.Id != nil {
				id = hit.Id.GetUuid()
			}

			results = append(results, SearchResult{
				ID:      id,
				Score:   hit.Score,
				Payload: payload,
			})
		}
		allResults = append(allResults, results)
	}

	return allResults, nil
}

// CountPoints returns the number of points in a collection
func (c *Client) CountPoints(ctx context.Context, collectionName string) (uint64, error) {
	if collectionName == "" {
		return 0, ErrEmptyCollection
	}

	resp, err := c.pointsClient.Count(ctx, &pb.CountPoints{
		CollectionName: collectionName,
	})

	if err != nil {
		return 0, WrapError(err, "failed to count points")
	}

	if resp.Result == nil {
		return 0, nil
	}

	return resp.Result.Count, nil
}
