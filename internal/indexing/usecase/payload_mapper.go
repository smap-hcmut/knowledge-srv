package usecase

import (
	"encoding/json"
	"knowledge-srv/internal/indexing"
)

type analyticsPayload struct {
	AnalyticsID           string                   `json:"analytics_id"`
	ProjectID             string                   `json:"project_id"`
	SourceID              string                   `json:"source_id"`
	Content               string                   `json:"content"`
	ContentCreatedAt      int64                    `json:"content_created_at"`
	IngestedAt            int64                    `json:"ingested_at"`
	Platform              string                   `json:"platform"`
	OverallSentiment      string                   `json:"overall_sentiment"`
	OverallSentimentScore float64                  `json:"overall_sentiment_score"`
	SentimentConfidence   float64                  `json:"sentiment_confidence"`
	Keywords              []string                 `json:"keywords"`
	RiskLevel             string                   `json:"risk_level"`
	RiskScore             float64                  `json:"risk_score"`
	RequiresAttention     bool                     `json:"requires_attention"`
	EngagementScore       float64                  `json:"engagement_score"`
	ViralityScore         float64                  `json:"virality_score"`
	InfluenceScore        float64                  `json:"influence_score"`
	ReachEstimate         int                      `json:"reach_estimate"`
	ContentQualityScore   float64                  `json:"content_quality_score"`
	IsSpam                bool                     `json:"is_spam"`
	IsBot                 bool                     `json:"is_bot"`
	Language              string                   `json:"language"`
	ToxicityScore         float64                  `json:"toxicity_score"`
	Aspects               []analyticsAspectPayload `json:"aspects,omitempty"`
	Metadata              analyticsMetadataPayload `json:"metadata"`
}

type analyticsAspectPayload struct {
	Aspect            string   `json:"aspect"`
	AspectDisplayName string   `json:"aspect_display_name"`
	Sentiment         string   `json:"sentiment"`
	SentimentScore    float64  `json:"sentiment_score"`
	Keywords          []string `json:"keywords"`
	ImpactScore       float64  `json:"impact_score"`
}

type analyticsMetadataPayload struct {
	Author            string                     `json:"author"`
	AuthorDisplayName string                     `json:"author_display_name"`
	AuthorUsername    string                     `json:"author_username,omitempty"`
	AuthorAvatar      string                     `json:"author_avatar,omitempty"`
	AuthorFollowers   int                        `json:"author_followers"`
	AuthorIsVerified  bool                       `json:"author_is_verified,omitempty"`
	Engagement        analyticsEngagementPayload `json:"engagement"`
	URL               string                     `json:"url,omitempty"`
	PostURL           string                     `json:"post_url,omitempty"`
	OriginalURL       string                     `json:"original_url,omitempty"`
	Permalink         string                     `json:"permalink,omitempty"`
	SourceURL         string                     `json:"source_url,omitempty"`
	WebURL            string                     `json:"web_url,omitempty"`
	CommentURL        string                     `json:"comment_url,omitempty"`
	ParentPostURL     string                     `json:"parent_post_url,omitempty"`
	VideoURL          string                     `json:"video_url,omitempty"`
	ContentType       string                     `json:"content_type,omitempty"`
	RootID            string                     `json:"root_id,omitempty"`
	ParentID          string                     `json:"parent_id,omitempty"`
	Hashtags          []string                   `json:"hashtags,omitempty"`
	Location          string                     `json:"location,omitempty"`
	PlatformMeta      map[string]any             `json:"platform_meta,omitempty"`
	Hierarchy         map[string]any             `json:"hierarchy,omitempty"`
}

type analyticsEngagementPayload struct {
	Views    int `json:"views"`
	Likes    int `json:"likes"`
	Comments int `json:"comments"`
	Shares   int `json:"shares"`
}

type insightPayload struct {
	ProjectID         string                 `json:"project_id"`
	CampaignID        string                 `json:"campaign_id"`
	UapID             string                 `json:"uap_id"`
	UapType           string                 `json:"uap_type"`
	UapMediaType      string                 `json:"uap_media_type"`
	Platform          string                 `json:"platform"`
	PublishedAt       string                 `json:"published_at"`
	URL               string                 `json:"url,omitempty"`
	PostURL           string                 `json:"post_url,omitempty"`
	OriginalURL       string                 `json:"original_url,omitempty"`
	Permalink         string                 `json:"permalink,omitempty"`
	SourceURL         string                 `json:"source_url,omitempty"`
	WebURL            string                 `json:"web_url,omitempty"`
	CommentURL        string                 `json:"comment_url,omitempty"`
	ParentPostURL     string                 `json:"parent_post_url,omitempty"`
	Author            string                 `json:"author,omitempty"`
	AuthorDisplayName string                 `json:"author_display_name,omitempty"`
	AuthorUsername    string                 `json:"author_username,omitempty"`
	AuthorAvatar      string                 `json:"author_avatar,omitempty"`
	ContentType       string                 `json:"content_type,omitempty"`
	RootID            string                 `json:"root_id,omitempty"`
	ParentID          string                 `json:"parent_id,omitempty"`
	PlatformMeta      map[string]interface{} `json:"platform_meta,omitempty"`
	Hierarchy         map[string]interface{} `json:"hierarchy,omitempty"`
	Content           string                 `json:"content"`
	ContentSummary    string                 `json:"content_summary"`
	ContextSummary    string                 `json:"context_summary,omitempty"`
	SentimentLabel    string                 `json:"sentiment_label"`
	SentimentScore    float64                `json:"sentiment_score"`
	Aspects           []insightAspectPayload `json:"aspects,omitempty"`
	Entities          []insightEntityPayload `json:"entities,omitempty"`
	ImpactScore       float64                `json:"impact_score"`
	RelevanceScore    float64                `json:"business_relevance_score"`
	RelevanceReasons  []string               `json:"business_relevance_reasons,omitempty"`
	Priority          string                 `json:"priority"`
	Likes             int                    `json:"likes"`
	Comments          int                    `json:"comments"`
	Shares            int                    `json:"shares"`
	Views             int                    `json:"views"`
}

type insightAspectPayload struct {
	Aspect   string `json:"aspect"`
	Polarity string `json:"polarity"`
}

type insightEntityPayload struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func (uc *implUseCase) payloadFromStruct(v interface{}) map[string]interface{} {
	data, err := json.Marshal(v)
	if err != nil {
		return map[string]interface{}{}
	}

	payload := make(map[string]interface{})
	if err := json.Unmarshal(data, &payload); err != nil {
		return map[string]interface{}{}
	}

	return payload
}

func mapAnalyticsAspects(aspects []indexing.Aspect) []analyticsAspectPayload {
	if len(aspects) == 0 {
		return nil
	}

	out := make([]analyticsAspectPayload, len(aspects))
	for i, aspect := range aspects {
		out[i] = analyticsAspectPayload{
			Aspect:            aspect.Aspect,
			AspectDisplayName: aspect.AspectDisplayName,
			Sentiment:         aspect.Sentiment,
			SentimentScore:    aspect.SentimentScore,
			Keywords:          aspect.Keywords,
			ImpactScore:       aspect.ImpactScore,
		}
	}

	return out
}

func mapInsightAspects(aspects []indexing.InsightAspectInput) []insightAspectPayload {
	if len(aspects) == 0 {
		return nil
	}

	out := make([]insightAspectPayload, len(aspects))
	for i, aspect := range aspects {
		out[i] = insightAspectPayload{
			Aspect:   aspect.Aspect,
			Polarity: aspect.Polarity,
		}
	}

	return out
}

func mapInsightEntities(entities []indexing.InsightEntityInput) []insightEntityPayload {
	if len(entities) == 0 {
		return nil
	}

	out := make([]insightEntityPayload, len(entities))
	for i, entity := range entities {
		out[i] = insightEntityPayload{
			Type:  entity.Type,
			Value: entity.Value,
		}
	}

	return out
}
