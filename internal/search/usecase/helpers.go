package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"knowledge-srv/internal/point"
	"knowledge-srv/internal/search"
)

// resolveCampaignProjects - Resolve campaign_id → project_ids (Tầng 2 cache)
func (uc *implUseCase) resolveCampaignProjects(ctx context.Context, campaignID string) ([]string, error) {
	// Check cache
	projectIDs, err := uc.cacheRepo.GetCampaignProjects(ctx, campaignID)
	if err == nil && len(projectIDs) > 0 {
		uc.l.Debugf(ctx, "search.usecase.resolveCampaignProjects: cache hit, %d projects", len(projectIDs))
		return projectIDs, nil
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

	// Save to cache
	if err := uc.cacheRepo.SaveCampaignProjects(ctx, campaignID, campaign.ProjectIDs); err != nil {
		uc.l.Warnf(ctx, "search.usecase.resolveCampaignProjects: cache save failed: %v", err)
	}

	return campaign.ProjectIDs, nil
}

// generateCacheKey - Generate Tầng 3 cache key
func (uc *implUseCase) generateCacheKey(input search.SearchInput) string {
	filterJSON, _ := json.Marshal(input.Filters)
	raw := fmt.Sprintf("%s:%s:%s:%d:%.2f", input.CampaignID, input.Query, string(filterJSON), input.Limit, input.MinScore)
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

	// Extract typed fields from payload
	if v, ok := r.Payload["content"].(string); ok {
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
	}
	if v, ok := r.Payload["overall_sentiment_score"].(float64); ok {
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
				if s, ok := m["sentiment"].(string); ok {
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
