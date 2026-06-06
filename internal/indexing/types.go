package indexing

import (
	"time"
)

const (
	MaxConcurrency            = 10
	MinContentLength          = 10
	MinBusinessRelevanceScore = 0.45
	EMBEDDING_ERROR           = "EMBEDDING_ERROR"
	QDRANT_ERROR              = "QDRANT_ERROR"
	DB_ERROR                  = "DB_ERROR"
	VALIDATION_ERROR          = "VALIDATION_ERROR"
	STATUS_INDEXED            = "INDEXED"
	STATUS_SKIPPED            = "SKIPPED"
	STATUS_FAILED             = "FAILED"
	STATUS_PENDING            = "PENDING"
)

type RetryFailedInput struct {
	MaxRetryCount int
	Limit         int
	ErrorTypes    []string
}

type ReconcileInput struct {
	StaleDuration time.Duration
	Limit         int
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

type IndexInsightInput struct {
	ProjectID           string
	CampaignID          string
	RunID               string
	InsightType         string
	Title               string
	Summary             string
	Confidence          float64
	AnalysisWindowStart string
	AnalysisWindowEnd   string
	SupportingMetrics   map[string]interface{}
	EvidenceReferences  []string
}

type IndexInsightOutput struct {
	PointID  string
	Duration time.Duration
}

type IndexDigestInput struct {
	ProjectID           string
	CampaignID          string
	RunID               string
	AnalysisWindowStart string
	AnalysisWindowEnd   string
	DomainOverlay       string
	Platform            string
	TotalMentions       int
	TopEntities         []TopEntityInput
	TopTopics           []TopTopicInput
	TopIssues           []TopIssueInput
}

type TopEntityInput struct {
	CanonicalEntityID string
	EntityName        string
	EntityType        string
	MentionCount      int
	MentionShare      float64
}

type TopTopicInput struct {
	TopicKey            string
	TopicLabel          string
	MentionCount        int
	MentionShare        float64
	BuzzScoreProxy      *float64
	QualityScore        *float64
	RepresentativeTexts []string
}

type TopIssueInput struct {
	IssueCategory      string
	MentionCount       int
	IssuePressureProxy float64
	SeverityMix        *SeverityMixInput
}

type SeverityMixInput struct {
	Low    float64
	Medium float64
	High   float64
}

type IndexDigestOutput struct {
	PointID  string
	Duration time.Duration
}

// IndexBatchInput - Input for direct payload indexing flow (Layer 3).
type IndexBatchInput struct {
	ProjectID  string
	CampaignID string
	Documents  []InsightMessageInput
}

// InsightMessageInput - Domain-level representation of one insight document.
type InsightMessageInput struct {
	Identity InsightIdentityInput
	Content  InsightContentInput
	NLP      InsightNLPInput
	Business InsightBusinessInput
	Source   InsightSourceInput
	RAG      bool
}

type InsightIdentityInput struct {
	UapID        string
	UapType      string
	UapMediaType string
	Platform     string
	PublishedAt  string
}

type InsightSourceInput struct {
	URL               string
	PostURL           string
	OriginalURL       string
	Permalink         string
	SourceURL         string
	WebURL            string
	CommentURL        string
	ParentPostURL     string
	Author            string
	AuthorDisplayName string
	AuthorUsername    string
	AuthorAvatar      string
	ContentType       string
	RootID            string
	ParentID          string
	PlatformMeta      map[string]interface{}
	Hierarchy         map[string]interface{}
}

type InsightContentInput struct {
	CleanText      string
	Summary        string
	ContextSummary string
}

type InsightNLPInput struct {
	Sentiment InsightSentimentInput
	Aspects   []InsightAspectInput
	Entities  []InsightEntityInput
}

type InsightSentimentInput struct {
	Label string
	Score float64
}

type InsightAspectInput struct {
	Aspect   string
	Polarity string
}

type InsightEntityInput struct {
	Type  string
	Value string
}

type InsightBusinessInput struct {
	Impact           InsightImpactInput
	RelevanceScore   float64
	RelevanceReasons []string
}

type InsightImpactInput struct {
	Engagement  InsightEngagementInput
	ImpactScore float64
	Priority    string
}

type InsightEngagementInput struct {
	Likes    int
	Comments int
	Shares   int
	Views    int
}

type IndexBatchOutput struct {
	ProjectID    string
	TotalRecords int
	Indexed      int
	Skipped      int
	Failed       int
	Duration     time.Duration
}
