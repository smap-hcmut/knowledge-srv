package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"knowledge-srv/internal/contentquality"
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

	// Step 3: Enrich query with campaign name for better semantic matching.
	// Generic analytical queries ("Tổng quan sentiment?") score very low against
	// brand-specific social content without this context prefix.
	// resolveCampaignProjects already cached the name, so this is effectively free.
	campaignName := uc.resolveCampaignName(ctx, input.CampaignID)
	enrichedQuery := input.Query
	if campaignName != "" {
		enrichedQuery = campaignName + ": " + input.Query
	}

	// Step 4: Embed enriched query (Via Embedding Domain)
	generateOutput, err := uc.embeddingUC.Generate(ctx, embedding.GenerateInput{
		Text: enrichedQuery,
	})
	if err != nil {
		uc.l.Errorf(ctx, "search.usecase.Search: Embedding generation failed: %v", err)
		return search.SearchOutput{}, fmt.Errorf("%w: %v", search.ErrEmbeddingFailed, err)
	}
	vector := generateOutput.Vector

	// Step 5: Build Qdrant filter (without project_id — implicit by collection)
	filter := uc.buildSearchFilter(nil, input.Filters)

	// Step 6: Search per-project Qdrant collections in parallel (server-side score filtering).
	// Over-fetch a bit so snapshot dedupe still leaves enough documents for the prompt.
	fetchLimit := limit * 3
	if fetchLimit > 50 {
		fetchLimit = 50
	}
	pointResults, err := uc.searchMultipleCollections(ctx, projectIDs, vector, filter, uint64(fetchLimit), float32(minScore))
	if err != nil {
		uc.l.Errorf(ctx, "search.usecase.Search: Multi-collection search failed: %v", err)
		return search.SearchOutput{}, fmt.Errorf("%w: %v", search.ErrSearchFailed, err)
	}

	// Step 7: Sort by score descending, collapse repeated snapshots of the same logical
	// post/UAP, then apply the final limit.
	sort.Slice(pointResults, func(i, j int) bool {
		return pointResults[i].Score > pointResults[j].Score
	})
	preDedupeCount := len(pointResults)
	pointResults = uc.dedupePointResults(pointResults)

	var candidates []search.SearchResult
	for _, r := range pointResults {
		mapped := uc.mapQdrantResult(r)
		if !isUsefulSearchResult(mapped) {
			continue
		}
		candidates = append(candidates, mapped)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return searchResultRankScore(candidates[i]) > searchResultRankScore(candidates[j])
	})

	results := candidates
	if len(results) > limit {
		results = results[:limit]
	}

	// Step 8: Hallucination control — NO relevant context flag
	noRelevantContext := len(results) == 0

	// Step 9: Build aggregations
	aggregations := uc.buildAggregations(results)

	// Step 10: Build output
	output := search.SearchOutput{
		Results:           results,
		TotalFound:        len(results),
		Aggregations:      aggregations,
		NoRelevantContext: noRelevantContext,
		CacheHit:          false,
		ProcessingTimeMs:  time.Since(startTime).Milliseconds(),
	}

	// Step 11: Cache results (Tầng 3)
	if data, err := json.Marshal(output); err == nil {
		if err := uc.cacheRepo.SaveSearchResults(ctx, cacheKey, data); err != nil {
			uc.l.Warnf(ctx, "search.usecase.Search: Failed to save cache: %v", err)
		}
	}

	uc.l.Infof(ctx, "search.usecase.Search: query=%q, enriched=%q, projects=%d, fetched=%d, deduped=%d, useful=%d, results=%d, no_context=%v, duration=%dms",
		input.Query, enrichedQuery, len(projectIDs), preDedupeCount, len(pointResults), len(candidates), len(results), noRelevantContext, output.ProcessingTimeMs)

	return output, nil
}

func isUsefulSearchResult(result search.SearchResult) bool {
	content := strings.TrimSpace(result.Content)
	if content == "" {
		return false
	}
	if contentquality.IsLowValueMarketingContent(content) {
		return false
	}
	biz := maxFloat(
		numberFromPayload(result.Metadata, "business_relevance_score"),
		numberFromNestedPayload(result.Metadata, "metadata", "business_relevance_score"),
	)
	if biz > 0 && biz < 0.30 && result.Score < 0.68 {
		return false
	}
	return true
}

func searchResultRankScore(result search.SearchResult) float64 {
	biz := maxFloat(
		numberFromPayload(result.Metadata, "business_relevance_score"),
		numberFromNestedPayload(result.Metadata, "metadata", "business_relevance_score"),
	)
	engagement := math.Log10(math.Max(result.EngagementScore, 0) + 1)
	contentCoverage := math.Min(float64(len([]rune(result.Content)))/260, 1)
	score := result.Score*10 + biz*2.2 + engagement*0.45 + contentCoverage*0.25
	switch strings.ToLower(strings.TrimSpace(result.OverallSentiment)) {
	case "negative":
		score += 0.20
	case "positive":
		score += 0.10
	}
	if sourceURLFromSearchPayload(result.Metadata) != "" {
		score += 0.15
	}
	if len(result.Aspects) > 0 {
		score += 0.20
	}
	return score
}

func sourceURLFromSearchPayload(payload map[string]interface{}) string {
	for _, value := range []string{
		stringFromNestedPayload(payload, "metadata", "comment_url"),
		stringFromNestedPayload(payload, "metadata", "original_url"),
		stringFromNestedPayload(payload, "metadata", "post_url"),
		stringFromNestedPayload(payload, "metadata", "permalink_url"),
		stringFromNestedPayload(payload, "metadata", "url"),
		stringFromPayload(payload, "comment_url"),
		stringFromPayload(payload, "original_url"),
		stringFromPayload(payload, "post_url"),
		stringFromPayload(payload, "permalink_url"),
		stringFromPayload(payload, "url"),
	} {
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
			return value
		}
	}
	return ""
}

func numberFromNestedPayload(payload map[string]interface{}, parentKey, childKey string) float64 {
	parent, ok := payload[parentKey]
	if !ok {
		return 0
	}
	obj, ok := parent.(map[string]interface{})
	if !ok {
		return 0
	}
	return numberFromPayload(obj, childKey)
}

func maxFloat(values ...float64) float64 {
	var max float64
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	return max
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
