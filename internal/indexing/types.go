package indexing

import (
	"time"
)

const (
	MaxConcurrency    = 10
	MinContentLength  = 10
	MinQualityScore   = 0.3
	EMBEDDING_ERROR   = "EMBEDDING_ERROR"
	QDRANT_ERROR      = "QDRANT_ERROR"
	DB_ERROR          = "DB_ERROR"
	VALIDATION_ERROR  = "VALIDATION_ERROR"
	DUPLICATE_CONTENT = "DUPLICATE_CONTENT"
	STATUS_INDEXED    = "INDEXED"
	STATUS_SKIPPED    = "SKIPPED"
	STATUS_FAILED     = "FAILED"
	STATUS_PENDING    = "PENDING"
)

type IndexInput struct {
	BatchID     string
	ProjectID   string
	FileURL     string
	RecordCount int
}

type RetryFailedInput struct {
	MaxRetryCount int
	Limit         int
	ErrorTypes    []string
}

type ReconcileInput struct {
	StaleDuration time.Duration
	Limit         int
}

type IndexOutput struct {
	BatchID       string
	TotalRecords  int
	Indexed       int
	Failed        int
	Skipped       int // Spam, bot, duplicate
	Duration      time.Duration
	FailedRecords []FailedRecord
}

type FailedRecord struct {
	AnalyticsID  string
	ErrorType    string
	ErrorMessage string
}

type RetryFailedOutput struct {
	TotalRetried int
	Succeeded    int
	Failed       int
	Duration     time.Duration
}

type ReconcileOutput struct {
	TotalChecked int
	Fixed        int
	Requeued     int
	Duration     time.Duration
}

type StatisticOutput struct {
	ProjectID      string
	TotalIndexed   int
	TotalFailed    int
	TotalPending   int
	LastIndexedAt  *time.Time
	AvgIndexTimeMs int
}

// AnalyticsPost - Cấu trúc của 1 record trong file JSONL từ Analytics Service
type AnalyticsPost struct {
	// Core Identity
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	SourceID  string `json:"source_id"`

	// UAP Core
	Content          string      `json:"content"`
	ContentCreatedAt time.Time   `json:"content_created_at"`
	IngestedAt       time.Time   `json:"ingested_at"`
	Platform         string      `json:"platform"`
	UAPMetadata      UAPMetadata `json:"uap_metadata"`

	// Sentiment
	OverallSentiment      string  `json:"overall_sentiment"`
	OverallSentimentScore float64 `json:"overall_sentiment_score"`
	SentimentConfidence   float64 `json:"sentiment_confidence"`
	SentimentExplanation  string  `json:"sentiment_explanation"`

	// ABSA
	Aspects []Aspect `json:"aspects"`

	// Keywords
	Keywords []string `json:"keywords"`

	// Risk
	RiskLevel         string       `json:"risk_level"`
	RiskScore         float64      `json:"risk_score"`
	RiskFactors       []RiskFactor `json:"risk_factors"`
	RequiresAttention bool         `json:"requires_attention"`
	AlertTriggered    bool         `json:"alert_triggered"`

	// Engagement
	EngagementScore float64 `json:"engagement_score"`
	ViralityScore   float64 `json:"virality_score"`
	InfluenceScore  float64 `json:"influence_score"`
	ReachEstimate   int     `json:"reach_estimate"`

	// Quality
	ContentQualityScore float64 `json:"content_quality_score"`
	IsSpam              bool    `json:"is_spam"`
	IsBot               bool    `json:"is_bot"`
	Language            string  `json:"language"`
	LanguageConfidence  float64 `json:"language_confidence"`
	ToxicityScore       float64 `json:"toxicity_score"`
	IsToxic             bool    `json:"is_toxic"`
}

// UAPMetadata - Metadata từ UAP
type UAPMetadata struct {
	Author            string             `json:"author"`
	AuthorDisplayName string             `json:"author_display_name"`
	AuthorFollowers   int                `json:"author_followers"`
	Engagement        EngagementMetadata `json:"engagement"`
	VideoURL          string             `json:"video_url,omitempty"`
	Hashtags          []string           `json:"hashtags,omitempty"`
	Location          string             `json:"location,omitempty"`
}

// EngagementMetadata - Engagement trong metadata
type EngagementMetadata struct {
	Views    int `json:"views"`
	Likes    int `json:"likes"`
	Comments int `json:"comments"`
	Shares   int `json:"shares"`
}

// Aspect - ABSA aspect
type Aspect struct {
	Aspect            string    `json:"aspect"`
	AspectDisplayName string    `json:"aspect_display_name"`
	Sentiment         string    `json:"sentiment"`
	SentimentScore    float64   `json:"sentiment_score"`
	Confidence        float64   `json:"confidence"`
	Keywords          []string  `json:"keywords"`
	Mentions          []Mention `json:"mentions"`
	ImpactScore       float64   `json:"impact_score"`
	Explanation       string    `json:"explanation"`
}

// Mention - Text mention trong aspect
type Mention struct {
	Text     string `json:"text"`
	StartPos int    `json:"start_pos"`
	EndPos   int    `json:"end_pos"`
}

// RiskFactor - Risk factor
type RiskFactor struct {
	Factor      string `json:"factor"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

// indexRecordResult - Result of processing single record
type IndexRecordResult struct {
	Status       string
	ErrorType    string
	ErrorMessage string
}
