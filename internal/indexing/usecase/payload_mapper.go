package usecase

import (
	"encoding/json"
	"knowledge-srv/internal/indexing"
)

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
