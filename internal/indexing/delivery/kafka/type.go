package kafka

import (
	"time"
)

// Topic & Group constants
const (
	// Layer 3
	TopicBatchCompleted   = "analytics.batch.completed"
	GroupIDBatchCompleted = "knowledge-indexing-batch"

	// Layer 2
	TopicInsightsPublished   = "analytics.insights.published"
	GroupIDInsightsPublished = "knowledge-indexing-insights"

	// Layer 1
	TopicReportDigest   = "analytics.report.digest"
	GroupIDReportDigest = "knowledge-indexing-digest"

	// DLQ topics
	TopicInsightsPublishedDLQ = "analytics.insights.published.dlq"
	TopicReportDigestDLQ      = "analytics.report.digest.dlq"
)

// LegacyBatchCompletedMessage is the old format (MinIO file reference).
type LegacyBatchCompletedMessage struct {
	BatchID     string    `json:"batch_id"`
	ProjectID   string    `json:"project_id"`
	CampaignID  string    `json:"campaign_id,omitempty"`
	FileURL     string    `json:"file_url"`
	RecordCount int       `json:"record_count"`
	CompletedAt time.Time `json:"completed_at"`
}

// BatchCompletedMessage is the new direct payload format.
type BatchCompletedMessage struct {
	ProjectID  string           `json:"project_id"`
	CampaignID string           `json:"campaign_id"`
	Documents  []InsightMessage `json:"documents"`
}

// InsightMessage is one document in documents[].
type InsightMessage struct {
	Identity InsightIdentity `json:"identity"`
	Content  InsightContent  `json:"content"`
	NLP      InsightNLP      `json:"nlp"`
	Business InsightBusiness `json:"business"`
	RAG      bool            `json:"rag"`
}

type InsightIdentity struct {
	UapID        string `json:"uap_id"`
	UapType      string `json:"uap_type"`
	UapMediaType string `json:"uap_media_type"`
	Platform     string `json:"platform"`
	PublishedAt  string `json:"published_at"`
}

type InsightContent struct {
	CleanText string `json:"clean_text"`
	Summary   string `json:"summary"`
}

type InsightNLP struct {
	Sentiment InsightSentiment `json:"sentiment"`
	Aspects   []InsightAspect  `json:"aspects"`
	Entities  []InsightEntity  `json:"entities"`
}

type InsightSentiment struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

type InsightAspect struct {
	Aspect   string `json:"aspect"`
	Polarity string `json:"polarity"`
}

type InsightEntity struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type InsightBusiness struct {
	Impact InsightImpact `json:"impact"`
}

type InsightImpact struct {
	Engagement  InsightEngagement `json:"engagement"`
	ImpactScore float64           `json:"impact_score"`
	Priority    string            `json:"priority"`
}

type InsightEngagement struct {
	Likes    int `json:"likes"`
	Comments int `json:"comments"`
	Shares   int `json:"shares"`
	Views    int `json:"views"`
}

// InsightsPublishedMessage - analytics.insights.published payload.
type InsightsPublishedMessage struct {
	ProjectID           string                 `json:"project_id"`
	CampaignID          string                 `json:"campaign_id"`
	RunID               string                 `json:"run_id"`
	InsightType         string                 `json:"insight_type"`
	Title               string                 `json:"title"`
	Summary             string                 `json:"summary"`
	Confidence          float64                `json:"confidence"`
	AnalysisWindowStart string                 `json:"analysis_window_start"`
	AnalysisWindowEnd   string                 `json:"analysis_window_end"`
	SupportingMetrics   map[string]interface{} `json:"supporting_metrics"`
	EvidenceReferences  []string               `json:"evidence_references"`
	ShouldIndex         bool                   `json:"should_index"`
}

// ReportDigestMessage - analytics.report.digest payload.
type ReportDigestMessage struct {
	ProjectID           string      `json:"project_id"`
	CampaignID          string      `json:"campaign_id"`
	RunID               string      `json:"run_id"`
	AnalysisWindowStart string      `json:"analysis_window_start"`
	AnalysisWindowEnd   string      `json:"analysis_window_end"`
	DomainOverlay       string      `json:"domain_overlay"`
	Platform            string      `json:"platform"`
	TotalMentions       int         `json:"total_mentions"`
	TopEntities         []TopEntity `json:"top_entities"`
	TopTopics           []TopTopic  `json:"top_topics"`
	TopIssues           []TopIssue  `json:"top_issues"`
	ShouldIndex         bool        `json:"should_index"`
}

type TopEntity struct {
	CanonicalEntityID string  `json:"canonical_entity_id"`
	EntityName        string  `json:"entity_name"`
	EntityType        string  `json:"entity_type"`
	MentionCount      int     `json:"mention_count"`
	MentionShare      float64 `json:"mention_share"`
}

type TopTopic struct {
	TopicKey            string   `json:"topic_key"`
	TopicLabel          string   `json:"topic_label"`
	MentionCount        int      `json:"mention_count"`
	MentionShare        float64  `json:"mention_share"`
	BuzzScoreProxy      *float64 `json:"buzz_score_proxy"`
	QualityScore        *float64 `json:"quality_score"`
	RepresentativeTexts []string `json:"representative_texts"`
}

type TopIssue struct {
	IssueCategory      string       `json:"issue_category"`
	MentionCount       int          `json:"mention_count"`
	IssuePressureProxy float64      `json:"issue_pressure_proxy"`
	SeverityMix        *SeverityMix `json:"severity_mix"`
}

type SeverityMix struct {
	Low    float64 `json:"low"`
	Medium float64 `json:"medium"`
	High   float64 `json:"high"`
}
