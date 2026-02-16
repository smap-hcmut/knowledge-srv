package http

import "knowledge-srv/internal/search"

// =====================================================
// Request DTOs
// =====================================================

type searchReq struct {
	CampaignID string           `json:"campaign_id" binding:"required"`
	Query      string           `json:"query" binding:"required,min=3,max=1000"`
	Filters    *searchFilterReq `json:"filters,omitempty"`
	Limit      int              `json:"limit,omitempty"`
	MinScore   float64          `json:"min_score,omitempty"`
}

type searchFilterReq struct {
	Sentiments    []string `json:"sentiments,omitempty"`
	Aspects       []string `json:"aspects,omitempty"`
	Platforms     []string `json:"platforms,omitempty"`
	DateFrom      *int64   `json:"date_from,omitempty"`
	DateTo        *int64   `json:"date_to,omitempty"`
	RiskLevels    []string `json:"risk_levels,omitempty"`
	MinEngagement *float64 `json:"min_engagement,omitempty"`
}

func (r searchReq) toInput() search.SearchInput {
	input := search.SearchInput{
		CampaignID: r.CampaignID,
		Query:      r.Query,
		Limit:      r.Limit,
		MinScore:   r.MinScore,
	}
	if r.Filters != nil {
		input.Filters = search.SearchFilters{
			Sentiments:    r.Filters.Sentiments,
			Aspects:       r.Filters.Aspects,
			Platforms:     r.Filters.Platforms,
			DateFrom:      r.Filters.DateFrom,
			DateTo:        r.Filters.DateTo,
			RiskLevels:    r.Filters.RiskLevels,
			MinEngagement: r.Filters.MinEngagement,
		}
	}
	return input
}

// =====================================================
// Response DTOs
// =====================================================

type searchResp struct {
	Results           []searchResultResp `json:"results"`
	TotalFound        int                `json:"total_found"`
	Aggregations      aggregationsResp   `json:"aggregations"`
	NoRelevantContext bool               `json:"no_relevant_context"`
	CacheHit          bool               `json:"cache_hit"`
	ProcessingTimeMs  int64              `json:"processing_time_ms"`
}

type searchResultResp struct {
	ID               string             `json:"id"`
	Score            float64            `json:"score"`
	Content          string             `json:"content"`
	ProjectID        string             `json:"project_id"`
	Platform         string             `json:"platform"`
	OverallSentiment string             `json:"overall_sentiment"`
	SentimentScore   float64            `json:"sentiment_score"`
	Aspects          []aspectResultResp `json:"aspects,omitempty"`
	Keywords         []string           `json:"keywords,omitempty"`
	RiskLevel        string             `json:"risk_level"`
	EngagementScore  float64            `json:"engagement_score"`
	ContentCreatedAt int64              `json:"content_created_at"`
}

type aspectResultResp struct {
	Aspect            string  `json:"aspect"`
	AspectDisplayName string  `json:"aspect_display_name"`
	Sentiment         string  `json:"sentiment"`
	SentimentScore    float64 `json:"sentiment_score"`
}

type aggregationsResp struct {
	BySentiment []sentimentAggResp `json:"by_sentiment"`
	ByAspect    []aspectAggResp    `json:"by_aspect"`
	ByPlatform  []platformAggResp  `json:"by_platform"`
}

type sentimentAggResp struct {
	Sentiment  string  `json:"sentiment"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type aspectAggResp struct {
	Aspect            string  `json:"aspect"`
	AspectDisplayName string  `json:"aspect_display_name"`
	Count             int     `json:"count"`
	AvgSentimentScore float64 `json:"avg_sentiment_score"`
}

type platformAggResp struct {
	Platform   string  `json:"platform"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

func (h *handler) newSearchResp(output search.SearchOutput) searchResp {
	resp := searchResp{
		TotalFound:        output.TotalFound,
		NoRelevantContext: output.NoRelevantContext,
		CacheHit:          output.CacheHit,
		ProcessingTimeMs:  output.ProcessingTimeMs,
	}

	// Map results
	resp.Results = make([]searchResultResp, len(output.Results))
	for i, r := range output.Results {
		result := searchResultResp{
			ID:               r.ID,
			Score:            r.Score,
			Content:          r.Content,
			ProjectID:        r.ProjectID,
			Platform:         r.Platform,
			OverallSentiment: r.OverallSentiment,
			SentimentScore:   r.SentimentScore,
			RiskLevel:        r.RiskLevel,
			EngagementScore:  r.EngagementScore,
			ContentCreatedAt: r.ContentCreatedAt,
			Keywords:         r.Keywords,
		}
		for _, a := range r.Aspects {
			result.Aspects = append(result.Aspects, aspectResultResp{
				Aspect:            a.Aspect,
				AspectDisplayName: a.AspectDisplayName,
				Sentiment:         a.Sentiment,
				SentimentScore:    a.SentimentScore,
			})
		}
		resp.Results[i] = result
	}

	// Map aggregations
	resp.Aggregations.BySentiment = make([]sentimentAggResp, len(output.Aggregations.BySentiment))
	for i, s := range output.Aggregations.BySentiment {
		resp.Aggregations.BySentiment[i] = sentimentAggResp{
			Sentiment: s.Sentiment, Count: s.Count, Percentage: s.Percentage,
		}
	}
	resp.Aggregations.ByAspect = make([]aspectAggResp, len(output.Aggregations.ByAspect))
	for i, a := range output.Aggregations.ByAspect {
		resp.Aggregations.ByAspect[i] = aspectAggResp{
			Aspect: a.Aspect, AspectDisplayName: a.AspectDisplayName,
			Count: a.Count, AvgSentimentScore: a.AvgSentimentScore,
		}
	}
	resp.Aggregations.ByPlatform = make([]platformAggResp, len(output.Aggregations.ByPlatform))
	for i, p := range output.Aggregations.ByPlatform {
		resp.Aggregations.ByPlatform[i] = platformAggResp{
			Platform: p.Platform, Count: p.Count, Percentage: p.Percentage,
		}
	}

	return resp
}
