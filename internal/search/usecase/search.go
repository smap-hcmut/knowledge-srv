package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/search"
	// For builders
)

// Search - Main search method
// Flow: check cache → resolve campaign → embed query → search Qdrant → filter by Score → aggregate → cache → return
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

	// Step 3: Embed query (Via Embedding Domain)
	generateOutput, err := uc.embeddingUC.Generate(ctx, embedding.GenerateInput{
		Text: input.Query,
	})
	if err != nil {
		uc.l.Errorf(ctx, "search.usecase.Search: Embedding generation failed: %v", err)
		return search.SearchOutput{}, fmt.Errorf("%w: %v", search.ErrEmbeddingFailed, err)
	}
	vector := generateOutput.Vector

	// Step 4: Build Qdrant filter
	filter := uc.buildSearchFilter(projectIDs, input.Filters)

	// Step 5: Search Qdrant (Via Point Domain)
	// Note: ScoreThreshold is implicit or handled by post-filtering
	pointResults, err := uc.pointUC.Search(ctx, point.SearchInput{
		Vector:         vector,
		Filter:         filter,
		Limit:          uint64(limit),
		WithPayload:    true,
		ScoreThreshold: 0,
	})
	if err != nil {
		uc.l.Errorf(ctx, "search.usecase.Search: Point search failed: %v", err)
		return search.SearchOutput{}, fmt.Errorf("%w: %v", search.ErrSearchFailed, err)
	}

	// Step 6: Filter by min score + map to domain results
	var results []search.SearchResult
	for _, r := range pointResults {
		if float64(r.Score) < minScore {
			continue
		}
		results = append(results, uc.mapQdrantResult(r))
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

	uc.l.Infof(ctx, "search.usecase.Search: query=%q, results=%d, no_context=%v, duration=%dms",
		input.Query, len(results), noRelevantContext, output.ProcessingTimeMs)

	return output, nil
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
