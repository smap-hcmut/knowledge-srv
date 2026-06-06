package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"knowledge-srv/internal/point"
	"knowledge-srv/internal/search"
)

// resolveCampaignProjects resolves campaign_id → project_ids visible to the
// caller. The cache layer (Tầng 2) still holds the full list per campaign;
// per-user filtering runs on top so two users querying the same campaign do
// not pollute each other's view. Previously this method returned every
// project in the campaign regardless of ownership, which let user A search /
// chat in user B's campaign as long as A knew the campaign_id.
func (uc *implUseCase) resolveCampaignProjects(ctx context.Context, userID, campaignID string) ([]string, error) {
	// Check cache (campaign-scoped, not user-scoped — we filter below).
	projectIDs, err := uc.cacheRepo.GetCampaignProjects(ctx, campaignID)
	if err == nil && len(projectIDs) > 0 {
		uc.l.Debugf(ctx, "search.usecase.resolveCampaignProjects: cache hit for campaign %s, %d projects", campaignID, len(projectIDs))
		return uc.filterProjectsByAccess(ctx, userID, projectIDs)
	}

	// Cache miss → call Project Service
	campaign, err := uc.projectSrv.GetCampaign(ctx, campaignID)
	if err != nil {
		uc.l.Errorf(ctx, "search.usecase.resolveCampaignProjects: GetCampaign failed: %v", err)
		return nil, fmt.Errorf("%w: %v", search.ErrCampaignNotFound, err)
	}

	if len(campaign.ProjectIDs) == 0 {
		return nil, search.ErrCampaignNoProjects
	}

	// Save project IDs to cache
	if err := uc.cacheRepo.SaveCampaignProjects(ctx, campaignID, campaign.ProjectIDs); err != nil {
		uc.l.Warnf(ctx, "search.usecase.resolveCampaignProjects: Failed to save project cache: %v", err)
	}

	// Also cache campaign name for query enrichment (best-effort)
	if campaign.Name != "" {
		if err := uc.cacheRepo.SaveCampaignName(ctx, campaignID, campaign.Name); err != nil {
			uc.l.Warnf(ctx, "search.usecase.resolveCampaignProjects: Failed to save name cache: %v", err)
		}
	}

	return uc.filterProjectsByAccess(ctx, userID, campaign.ProjectIDs)
}

// filterProjectsByAccess keeps only project IDs the caller is allowed to read.
// Returns search.ErrCampaignNoProjects when the filter removes every project
// so the caller treats the result the same as an empty campaign — knowledge-srv
// returns a generic "no scope" error to the client rather than disclosing
// which projects exist.
func (uc *implUseCase) filterProjectsByAccess(ctx context.Context, userID string, projectIDs []string) ([]string, error) {
	if strings.TrimSpace(userID) == "" {
		// No user context = internal call (e.g. health probe). Be conservative
		// and reject; routes that need a real user must populate the scope.
		return nil, search.ErrCampaignNoProjects
	}

	allowed := make([]string, 0, len(projectIDs))
	for _, projectID := range projectIDs {
		ok, err := uc.projectSrv.ValidateProjectAccess(ctx, userID, projectID)
		if err != nil {
			uc.l.Warnf(ctx, "search.usecase.resolveCampaignProjects: ValidateProjectAccess(%s,%s) failed: %v", userID, projectID, err)
			continue
		}
		if ok {
			allowed = append(allowed, projectID)
		}
	}
	if len(allowed) == 0 {
		return nil, search.ErrCampaignNoProjects
	}
	return allowed, nil
}

// resolveCampaignName returns the campaign display name for query enrichment.
// Always a cache hit after resolveCampaignProjects has run in the same request.
// Returns empty string on any error (enrichment is best-effort).
func (uc *implUseCase) resolveCampaignName(ctx context.Context, campaignID string) string {
	name, err := uc.cacheRepo.GetCampaignName(ctx, campaignID)
	if err == nil && name != "" {
		return name
	}
	// Fallback: fetch directly if cache was cold (e.g. first request after restart)
	campaign, err := uc.projectSrv.GetCampaign(ctx, campaignID)
	if err != nil {
		return ""
	}
	if campaign.Name != "" {
		_ = uc.cacheRepo.SaveCampaignName(ctx, campaignID, campaign.Name)
	}
	return campaign.Name
}

// generateCacheKey - Generate Tầng 3 cache key
func (uc *implUseCase) generateCacheKey(input search.SearchInput) string {
	filterJSON, _ := json.Marshal(input.Filters)
	raw := fmt.Sprintf("v3:%s:%s:%s:%d:%.2f", input.CampaignID, input.Query, string(filterJSON), input.Limit, input.MinScore)
	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("search:%s:%x", input.CampaignID, hash)
}

// mapQdrantResult - Map Point SearchOutput → Domain SearchResult
func (uc *implUseCase) mapQdrantResult(r point.SearchOutput) search.SearchResult {
	result := search.SearchResult{
		ID:       r.ID,
		Score:    float64(r.Score),
		Metadata: r.Payload,
	}

	// Extract typed fields from payload.
	// Keep both legacy analytics field names and current direct insight field names
	// readable because old Qdrant collections may still contain pre-migration points.
	if v, ok := r.Payload["content"].(string); ok {
		result.Content = v
	} else if v, ok := r.Payload["content_summary"].(string); ok {
		result.Content = v
	}
	if v, ok := r.Payload["project_id"].(string); ok {
		result.ProjectID = v
	}
	if v, ok := r.Payload["platform"].(string); ok {
		result.Platform = v
	}
	if v, ok := r.Payload["overall_sentiment"].(string); ok {
		result.OverallSentiment = v
	} else if v, ok := r.Payload["sentiment_label"].(string); ok {
		result.OverallSentiment = v
	}
	if v, ok := r.Payload["overall_sentiment_score"].(float64); ok {
		result.SentimentScore = v
	} else if v, ok := r.Payload["sentiment_score"].(float64); ok {
		result.SentimentScore = v
	}
	if v, ok := r.Payload["risk_level"].(string); ok {
		result.RiskLevel = v
	}
	if v, ok := r.Payload["engagement_score"].(float64); ok {
		result.EngagementScore = v
	}
	if v, ok := r.Payload["content_created_at"].(float64); ok {
		result.ContentCreatedAt = int64(v)
	}
	if v, ok := r.Payload["keywords"].([]interface{}); ok {
		for _, k := range v {
			if s, ok := k.(string); ok {
				result.Keywords = append(result.Keywords, s)
			}
		}
	}
	// Aspects extraction
	if v, ok := r.Payload["aspects"].([]interface{}); ok {
		for _, a := range v {
			if m, ok := a.(map[string]interface{}); ok {
				aspect := search.AspectResult{}
				if s, ok := m["aspect"].(string); ok {
					aspect.Aspect = s
				}
				if s, ok := m["aspect_display_name"].(string); ok {
					aspect.AspectDisplayName = s
				}
				// Legacy points use "sentiment"; direct insight points use "polarity".
				if s, ok := m["sentiment"].(string); ok {
					aspect.Sentiment = s
				} else if s, ok := m["polarity"].(string); ok {
					aspect.Sentiment = s
				}
				if f, ok := m["sentiment_score"].(float64); ok {
					aspect.SentimentScore = f
				}
				result.Aspects = append(result.Aspects, aspect)
			}
		}
	}

	return result
}

// dedupePointResults collapses multiple snapshots of the same logical post/UAP.
// Keep the first item because callers already sort by descending relevance score.
func (uc *implUseCase) dedupePointResults(results []point.SearchOutput) []point.SearchOutput {
	if len(results) == 0 {
		return nil
	}

	deduped := make([]point.SearchOutput, 0, len(results))
	seen := make(map[string]struct{}, len(results))

	for _, result := range results {
		key := dedupeKeyForPointResult(result)
		if key == "" {
			deduped = append(deduped, result)
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, result)
	}

	return deduped
}

func dedupeKeyForPointResult(result point.SearchOutput) string {
	payload := result.Payload
	platform := normalizeDedupeValue(stringFromPayload(payload, "platform"))
	if platform == "" {
		platform = "unknown"
	}

	if uapID := normalizeDedupeValue(stringFromPayload(payload, "uap_id")); uapID != "" {
		return fmt.Sprintf("%s|uap|%s", platform, uapID)
	}

	content := normalizeDedupeValue(firstNonEmpty(
		stringFromPayload(payload, "content"),
		stringFromPayload(payload, "content_summary"),
	))
	author := normalizeDedupeValue(firstNonEmpty(
		stringFromNestedPayload(payload, "metadata", "author_username"),
		stringFromNestedPayload(payload, "metadata", "author"),
		stringFromNestedPayload(payload, "metadata", "author_display_name"),
	))
	createdAt := normalizeDedupeValue(firstNonEmpty(
		stringFromPayload(payload, "published_at"),
		fmt.Sprintf("%.0f", numberFromPayload(payload, "content_created_at")),
	))
	if content != "" {
		return fmt.Sprintf("%s|content|%s|%s|%s", platform, author, createdAt, content)
	}

	if sourceID := normalizeDedupeValue(stringFromPayload(payload, "source_id")); sourceID != "" {
		return fmt.Sprintf("%s|source|%s", platform, sourceID)
	}

	contentURL := normalizeDedupeValue(firstNonEmpty(
		stringFromNestedPayload(payload, "metadata", "url"),
		stringFromNestedPayload(payload, "metadata", "video_url"),
	))
	if contentURL != "" {
		return fmt.Sprintf("%s|url|%s", platform, contentURL)
	}

	return ""
}

func numberFromPayload(payload map[string]interface{}, key string) float64 {
	v, ok := payload[key]
	if !ok {
		return 0
	}
	f, ok := v.(float64)
	if !ok {
		return 0
	}
	return f
}

func stringFromPayload(payload map[string]interface{}, key string) string {
	v, ok := payload[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func stringFromNestedPayload(payload map[string]interface{}, parentKey, childKey string) string {
	parent, ok := payload[parentKey]
	if !ok {
		return ""
	}
	obj, ok := parent.(map[string]interface{})
	if !ok {
		return ""
	}
	v, ok := obj[childKey]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeDedupeValue(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}
	return strings.Join(strings.Fields(trimmed), " ")
}

// buildAggregations - Tổng hợp thống kê từ results
func (uc *implUseCase) buildAggregations(results []search.SearchResult) search.Aggregations {
	total := len(results)
	if total == 0 {
		return search.Aggregations{}
	}

	// Sentiment aggregation
	sentimentCounts := make(map[string]int)
	for _, r := range results {
		if r.OverallSentiment != "" {
			sentimentCounts[r.OverallSentiment]++
		}
	}
	var bySentiment []search.SentimentAgg
	for s, c := range sentimentCounts {
		bySentiment = append(bySentiment, search.SentimentAgg{
			Sentiment:  s,
			Count:      c,
			Percentage: float64(c) / float64(total) * 100,
		})
	}

	// Platform aggregation
	platformCounts := make(map[string]int)
	for _, r := range results {
		if r.Platform != "" {
			platformCounts[r.Platform]++
		}
	}
	var byPlatform []search.PlatformAgg
	for p, c := range platformCounts {
		byPlatform = append(byPlatform, search.PlatformAgg{
			Platform:   p,
			Count:      c,
			Percentage: float64(c) / float64(total) * 100,
		})
	}

	// Aspect aggregation
	aspectData := make(map[string]struct {
		DisplayName string
		Count       int
		TotalScore  float64
	})
	for _, r := range results {
		for _, a := range r.Aspects {
			d := aspectData[a.Aspect]
			d.DisplayName = a.AspectDisplayName
			d.Count++
			d.TotalScore += a.SentimentScore
			aspectData[a.Aspect] = d
		}
	}
	var byAspect []search.AspectAgg
	for name, d := range aspectData {
		byAspect = append(byAspect, search.AspectAgg{
			Aspect:            name,
			AspectDisplayName: d.DisplayName,
			Count:             d.Count,
			AvgSentimentScore: d.TotalScore / float64(d.Count),
		})
	}

	return search.Aggregations{
		BySentiment: bySentiment,
		ByAspect:    byAspect,
		ByPlatform:  byPlatform,
	}
}
