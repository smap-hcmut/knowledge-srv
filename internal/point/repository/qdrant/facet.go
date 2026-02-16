package qdrant

import (
	"context"
	"fmt"

	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
)

func (r *implRepository) Facet(ctx context.Context, input repository.FacetOptions) ([]point.FacetOutput, error) {

	pkgResults, err := r.client.Facet(ctx, collectionName, input.Key, input.Limit, input.Filter)
	if err != nil {
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
