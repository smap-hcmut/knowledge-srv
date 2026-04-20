package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/search"
	pkgQdrant "knowledge-srv/pkg/qdrant"

	"golang.org/x/sync/errgroup"
)

// Search - Main search method
// Flow: check cache → resolve campaign → embed query → search per-project Qdrant collections → filter by Score → aggregate → cache → return
func (uc *implUseCase) Search(ctx context.Context, sc model.Scope, input search.SearchInput) (search.SearchOutput, error) {
	startTime := time.Now()

	// Step 0: Validate input
	if err := uc.validateInput(input); err != nil {
		return search.SearchOutput{}, err
	}

	// Apply defaults
	limit := input.Limit
	if limit <= 0 {
		limit = search.MaxResults
	}
	if limit > 50 {
		limit = 50
	}
	minScore := input.MinScore
	if minScore <= 0 {
		minScore = search.MinScore
	}

	// Step 1: Check Tầng 3 — Search Results Cache
	cacheKey := uc.generateCacheKey(input)
	cachedData, err := uc.cacheRepo.GetSearchResults(ctx, cacheKey)
	if err == nil && cachedData != nil {
		var cached search.SearchOutput
		if err := json.Unmarshal(cachedData, &cached); err == nil {
			cached.CacheHit = true
			cached.ProcessingTimeMs = time.Since(startTime).Milliseconds()
			uc.l.Debugf(ctx, "search.usecase.Search: cache hit for key %s", cacheKey)
			return cached, nil
		}
	}

	// Step 2: Resolve campaign → project_ids (Tầng 2 cache)
	projectIDs, err := uc.resolveCampaignProjects(ctx, input.CampaignID)
	if err != nil {
		return search.SearchOutput{}, err
	}
	if len(projectIDs) == 0 {
		return search.SearchOutput{
			NoRelevantContext: true,
			ProcessingTimeMs:  time.Since(startTime).Milliseconds(),
		}, nil
	}

	// Step 3: Embed query (Via Embedding Domain)
	generateOutput, err := uc.embeddingUC.Generate(ctx, embedding.GenerateInput{
		Text: input.Query,
	})
	if err != nil {
		uc.l.Errorf(ctx, "search.usecase.Search: Embedding generation failed: %v", err)
		return search.SearchOutput{}, fmt.Errorf("%w: %v", search.ErrEmbeddingFailed, err)
	}
	vector := generateOutput.Vector

	// Step 4: Build Qdrant filter (without project_id — implicit by collection)
	filter := uc.buildSearchFilter(nil, input.Filters)

	// Step 5: Search per-project Qdrant collections in parallel (server-side score filtering)
	pointResults, err := uc.searchMultipleCollections(ctx, projectIDs, vector, filter, uint64(limit), float32(minScore))
	if err != nil {
		uc.l.Errorf(ctx, "search.usecase.Search: Multi-collection search failed: %v", err)
		return search.SearchOutput{}, fmt.Errorf("%w: %v", search.ErrSearchFailed, err)
	}

	// Step 6: Sort by score descending and apply limit (Qdrant already filtered by minScore server-side)
	sort.Slice(pointResults, func(i, j int) bool {
		return pointResults[i].Score > pointResults[j].Score
	})

	var results []search.SearchResult
	for _, r := range pointResults {
		results = append(results, uc.mapQdrantResult(r))
		if len(results) >= limit {
			break
		}
	}

	// Step 7: Hallucination control — NO relevant context flag
	noRelevantContext := len(results) == 0

	// Step 8: Build aggregations
	aggregations := uc.buildAggregations(results)

	// Step 9: Build output
	output := search.SearchOutput{
		Results:           results,
		TotalFound:        len(results),
		Aggregations:      aggregations,
		NoRelevantContext: noRelevantContext,
		CacheHit:          false,
		ProcessingTimeMs:  time.Since(startTime).Milliseconds(),
	}

	// Step 10: Cache results (Tầng 3)
	if data, err := json.Marshal(output); err == nil {
		if err := uc.cacheRepo.SaveSearchResults(ctx, cacheKey, data); err != nil {
			uc.l.Warnf(ctx, "search.usecase.Search: Failed to save cache: %v", err)
		}
	}

	uc.l.Infof(ctx, "search.usecase.Search: query=%q, projects=%d, results=%d, no_context=%v, duration=%dms",
		input.Query, len(projectIDs), len(results), noRelevantContext, output.ProcessingTimeMs)

	return output, nil
}

// searchMultipleCollections searches across per-project Qdrant collections in parallel.
// Non-existent collections are silently skipped (project may not have indexed data yet).
func (uc *implUseCase) searchMultipleCollections(
	ctx context.Context,
	projectIDs []string,
	vector []float32,
	filter *point.Filter,
	limit uint64,
	scoreThreshold float32,
) ([]point.SearchOutput, error) {
	var (
		allResults []point.SearchOutput
		mu         sync.Mutex
	)

	g, gCtx := errgroup.WithContext(ctx)

	for _, pid := range projectIDs {
		collectionName := point.CollectionForProject(pid)
		g.Go(func() error {
			results, err := uc.pointUC.Search(gCtx, point.SearchInput{
				CollectionName: collectionName,
				Vector:         vector,
				Filter:         filter,
				Limit:          limit,
				WithPayload:    true,
				ScoreThreshold: scoreThreshold,
			})
			if err != nil {
				// Skip non-existent collections (project may not have data yet)
				if isCollectionNotFoundError(err) {
					uc.l.Debugf(gCtx, "search.usecase.searchMultipleCollections: collection %s not found, skipping", collectionName)
					return nil
				}
				return fmt.Errorf("search collection %s: %w", collectionName, err)
			}

			mu.Lock()
			allResults = append(allResults, results...)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return allResults, nil
}

// isCollectionNotFoundError checks if the Qdrant error indicates a missing collection.
func isCollectionNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, pkgQdrant.ErrCollectionNotFound) {
		return true
	}
	// Fallback: string match for errors that may not be wrapped with the sentinel.
	msg := err.Error()
	return strings.Contains(msg, "Not found") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "doesn't exist") ||
		strings.Contains(msg, "does not exist")
}

// validateInput - Validate search input
func (uc *implUseCase) validateInput(input search.SearchInput) error {
	if input.CampaignID == "" {
		return search.ErrCampaignNotFound
	}
	if len(input.Query) < search.MinQueryLength {
		return search.ErrQueryTooShort
	}
	if len(input.Query) > search.MaxQueryLength {
		return search.ErrQueryTooLong
	}
	return nil
}
