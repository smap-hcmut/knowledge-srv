package usecase

import (
	"context"
	"fmt"

	// Using sync.Mutex for map writes if needed, or just specific vars
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/search"

	pb "github.com/qdrant/go-client/qdrant"
	"golang.org/x/sync/errgroup"
)

func (uc *implUseCase) Aggregate(ctx context.Context, sc model.Scope, input search.AggregateInput) (search.AggregateOutput, error) {
	// Step 1: Resolve campaign -> projects
	projectIDs, err := uc.resolveCampaignProjects(ctx, input.CampaignID)
	if err != nil {
		return search.AggregateOutput{}, err
	}

	// Step 2: Build Base Filter (Project IN [...])
	baseFilter := &pb.Filter{
		Must: []*pb.Condition{
			{
				ConditionOneOf: &pb.Condition_Field{
					Field: &pb.FieldCondition{
						Key: "project_id",
						Match: &pb.Match{
							MatchValue: &pb.Match_Keywords{
								Keywords: &pb.RepeatedStrings{Strings: projectIDs},
							},
						},
					},
				},
			},
		},
	}

	var output search.AggregateOutput
	var (
		sentimentRes []point.FacetOutput
		platformRes  []point.FacetOutput
		aspectRes    []point.FacetOutput
		totalDocs    uint64
	)

	g, ctx := errgroup.WithContext(ctx)

	// Task 1: Total Docs
	g.Go(func() error {
		count, err := uc.pointUC.Count(ctx, point.CountInput{Filter: baseFilter})
		if err != nil {
			return err
		}
		totalDocs = count
		return nil
	})

	// Task 2: Sentiment Breakdown
	g.Go(func() error {
		res, err := uc.pointUC.Facet(ctx, point.FacetInput{
			Key:    "overall_sentiment",
			Filter: baseFilter,
			Limit:  10,
		})
		if err != nil {
			return err
		}
		sentimentRes = res
		return nil
	})

	// Task 3: Platform Breakdown
	g.Go(func() error {
		res, err := uc.pointUC.Facet(ctx, point.FacetInput{
			Key:    "platform",
			Filter: baseFilter,
			Limit:  10,
		})
		if err != nil {
			return err
		}
		platformRes = res
		return nil
	})

	// Task 4: Top Negative Aspects
	// Filter: Base + overall_sentiment = NEGATIVE (Approximation)
	// Ideal: Nested filter on aspects.sentiment. But Facet on nested fields is complex.
	// We'll use overall_sentiment to find "Negative Posts" and see what aspects they talk about.
	g.Go(func() error {
		negFilter := &pb.Filter{
			Must: append(baseFilter.Must, &pb.Condition{
				ConditionOneOf: &pb.Condition_Field{
					Field: &pb.FieldCondition{
						Key: "overall_sentiment",
						Match: &pb.Match{
							MatchValue: &pb.Match_Keyword{Keyword: "NEGATIVE"},
						},
					},
				},
			}),
		}
		// Facet on "aspects.aspect" (assuming indexing supports this path)
		res, err := uc.pointUC.Facet(ctx, point.FacetInput{
			Key:    "aspects.aspect",
			Filter: negFilter,
			Limit:  5,
		})
		if err != nil {
			// If aspects are not indexed for faceting, this might fail or return empty.
			// Log warning and ignore? Or return error?
			// For now, return error to surface config issues.
			return fmt.Errorf("failed to facet aspects: %w", err)
		}
		aspectRes = res
		return nil
	})

	if err := g.Wait(); err != nil {
		uc.l.Errorf(ctx, "search.usecase.Aggregate: One or more tasks failed: %v", err)
		return search.AggregateOutput{}, err
	}

	// Assembly Output
	output.TotalDocs = totalDocs
	output.SentimentBreakdown = make(map[string]uint64)
	output.PlatformBreakdown = make(map[string]uint64)

	for _, s := range sentimentRes {
		output.SentimentBreakdown[s.Value] = s.Count
	}
	for _, p := range platformRes {
		output.PlatformBreakdown[p.Value] = p.Count
	}
	for _, a := range aspectRes {
		output.TopNegativeAspects = append(output.TopNegativeAspects, search.AspectCount{
			Aspect: a.Value,
			Count:  a.Count,
		})
	}

	return output, nil
}
