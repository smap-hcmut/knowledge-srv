package qdrant

import (
	"context"

	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Facet performs a facet request to get value counts.
func (c *qdrantImpl) Facet(ctx context.Context, collectionName string, key string, limit uint64, filter *pb.Filter) ([]FacetResult, error) {
	if collectionName == "" {
		return nil, ErrEmptyCollection
	}
	if key == "" {
		return nil, ErrEmptyKey
	}

	// Try using the PointsClient.Facet method if available
	// Note: If the generated client is too old, this will fail to compile.
	// We assume standard Qdrant client support for Facet (v1.10+).

	resp, err := c.pointsClient.Facet(ctx, &pb.FacetCounts{
		CollectionName: collectionName,
		Key:            key,
		Filter:         filter,
		Limit:          &limit,
		Exact:          new(bool), // Default false for performance
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrCollectionNotFound
		}
		return nil, WrapError(err, "failed to get facets")
	}

	results := make([]FacetResult, 0, len(resp.Hits))
	for _, hit := range resp.Hits {
		var val interface{}
		// FacetValue oneof: string_value, integer_value, etc.
		if v := hit.Value.GetStringValue(); v != "" {
			val = v
		} else {
			val = hit.Value.GetIntegerValue()
		}

		results = append(results, FacetResult{
			Value: val,
			Count: hit.Count,
		})
	}

	return results, nil
}
