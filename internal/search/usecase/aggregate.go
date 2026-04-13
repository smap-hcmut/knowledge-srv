package usecase

import (
	"context"
	"fmt"
	"sync"

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
	if len(projectIDs) == 0 {
		return search.AggregateOutput{
			SentimentBreakdown: make(map[string]uint64),
			PlatformBreakdown:  make(map[string]uint64),
		}, nil
	}

	// Step 2: Query per-project collections in parallel, merge results
	var (
		totalDocs    uint64
		sentimentMap = make(map[string]uint64)
		platformMap  = make(map[string]uint64)
		aspectMap    = make(map[string]uint64)
		mu           sync.Mutex
	)

	g, gCtx := errgroup.WithContext(ctx)

	for _, pid := range projectIDs {
		collectionName := point.CollectionForProject(pid)
		g.Go(func() error {
			return uc.aggregateCollection(gCtx, collectionName, &totalDocs, sentimentMap, platformMap, aspectMap, &mu)
		})
	}

	if err := g.Wait(); err != nil {
		uc.l.Errorf(ctx, "search.usecase.Aggregate: One or more tasks failed: %v", err)
		return search.AggregateOutput{}, err
	}

	// Assembly Output
	var output search.AggregateOutput
	output.TotalDocs = totalDocs
	output.SentimentBreakdown = sentimentMap
	output.PlatformBreakdown = platformMap

	for aspect, count := range aspectMap {
		output.TopNegativeAspects = append(output.TopNegativeAspects, search.AspectCount{
			Aspect: aspect,
			Count:  count,
		})
	}

	return output, nil
}

// aggregateCollection runs count and facet queries on a single Qdrant collection,
// merging results into the shared maps. Non-existent collections are skipped.
func (uc *implUseCase) aggregateCollection(
	ctx context.Context,
	collectionName string,
	totalDocs *uint64,
	sentimentMap, platformMap, aspectMap map[string]uint64,
	mu *sync.Mutex,
) error {
	// No project_id filter needed — collection is already per-project
	emptyFilter := &pb.Filter{}

	var (
		colTotal   uint64
		sentimentR []point.FacetOutput
		platformR  []point.FacetOutput
		aspectR    []point.FacetOutput
	)

	g, gCtx := errgroup.WithContext(ctx)

	// Count
	g.Go(func() error {
		count, err := uc.pointUC.Count(gCtx, point.CountInput{
			CollectionName: collectionName,
			Filter:         emptyFilter,
		})
		if err != nil {
			if isCollectionNotFoundError(err) {
				return nil
			}
			return err
		}
		colTotal = count
		return nil
	})

	// Sentiment
	g.Go(func() error {
		res, err := uc.pointUC.Facet(gCtx, point.FacetInput{
			CollectionName: collectionName,
			Key:            "overall_sentiment",
			Filter:         emptyFilter,
			Limit:          10,
		})
		if err != nil {
			if isCollectionNotFoundError(err) {
				return nil
			}
			return err
		}
		sentimentR = res
		return nil
	})

	// Platform
	g.Go(func() error {
		res, err := uc.pointUC.Facet(gCtx, point.FacetInput{
			CollectionName: collectionName,
			Key:            "platform",
			Filter:         emptyFilter,
			Limit:          10,
		})
		if err != nil {
			if isCollectionNotFoundError(err) {
				return nil
			}
			return err
		}
		platformR = res
		return nil
	})

	// Negative aspects
	g.Go(func() error {
		negFilter := &pb.Filter{
			Must: []*pb.Condition{
				{
					ConditionOneOf: &pb.Condition_Field{
						Field: &pb.FieldCondition{
							Key: "overall_sentiment",
							Match: &pb.Match{
								MatchValue: &pb.Match_Keyword{Keyword: "NEGATIVE"},
							},
						},
					},
				},
			},
		}
		res, err := uc.pointUC.Facet(gCtx, point.FacetInput{
			CollectionName: collectionName,
			Key:            "aspects.aspect",
			Filter:         negFilter,
			Limit:          5,
		})
		if err != nil {
			if isCollectionNotFoundError(err) {
				return nil
			}
			return fmt.Errorf("failed to facet aspects in %s: %w", collectionName, err)
		}
		aspectR = res
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}

	// Merge results into shared maps
	mu.Lock()
	defer mu.Unlock()

	*totalDocs += colTotal
	for _, s := range sentimentR {
		sentimentMap[s.Value] += s.Count
	}
	for _, p := range platformR {
		platformMap[p.Value] += p.Count
	}
	for _, a := range aspectR {
		aspectMap[a.Value] += a.Count
	}

	return nil
}
