package qdrant

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pkgQdrant "knowledge-srv/pkg/qdrant"

	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
)

func (r *implRepository) Facet(ctx context.Context, input repository.FacetOptions) ([]point.FacetOutput, error) {

	pkgResults, err := r.client.Facet(ctx, input.CollectionName, input.Key, input.Limit, input.Filter)
	if err != nil {
		if errors.Is(err, pkgQdrant.ErrCollectionNotFound) {
			// Collection doesn't exist yet (no data indexed) — not an error, just empty.
			return nil, err
		}
		if strings.Contains(err.Error(), "No appropriate index for faceting") {
			// Some legacy collections do not have every optional payload index yet.
			// Callers may intentionally fall back when a facet is unavailable.
			return nil, err
		}
		r.l.Errorf(ctx, "point.repository.qdrant.Facet: Failed to facet points: %v", err)
		return nil, err
	}

	results := make([]point.FacetOutput, len(pkgResults))
	for i, pr := range pkgResults {
		var valStr string
		switch v := pr.Value.(type) {
		case string:
			valStr = v
		case int64:
			valStr = fmt.Sprintf("%d", v)
		case int:
			valStr = fmt.Sprintf("%d", v)
		case float64:
			valStr = fmt.Sprintf("%.2f", v)
		default:
			if v != nil {
				valStr = fmt.Sprintf("%v", v)
			}
		}

		results[i] = point.FacetOutput{
			Value: valStr,
			Count: pr.Count,
		}
	}

	return results, nil
}
