# M2: Type Definitions — Kafka Message Structs + Domain Types

**Prerequisite**: M1 phải done.

**Goal**: Khai báo tất cả Go structs cho 3 contracts mới. Không có runtime behavior thay đổi — chỉ thêm types.

**Risk**: Rất thấp — purely additive, không break gì.

---

## Files cần thay đổi

### 1. `internal/indexing/delivery/kafka/type.go`

**REPLACE toàn bộ file** (giữ existing `BatchCompletedMessage` và rename sang `LegacyBatchCompletedMessage`, thêm version mới):

```go
package kafka

import (
	"time"
)

// ── Topic & Group constants ──────────────────────────────────────────────────

const (
	// Layer 3 (existing)
	TopicBatchCompleted   = "analytics.batch.completed"
	GroupIDBatchCompleted = "knowledge-indexing-batch"

	// Layer 2 (NEW)
	TopicInsightsPublished   = "analytics.insights.published"
	GroupIDInsightsPublished = "knowledge-indexing-insights"

	// Layer 1 (NEW)
	TopicReportDigest   = "analytics.report.digest"
	GroupIDReportDigest = "knowledge-indexing-digest"

	// DLQ topics (NEW)
	TopicInsightsPublishedDLQ = "analytics.insights.published.dlq"
	TopicReportDigestDLQ      = "analytics.report.digest.dlq"

	// Qdrant collection names (NEW)
	CollectionMacroInsights = "macro_insights"
)

// ── Layer 3: analytics.batch.completed ──────────────────────────────────────

// LegacyBatchCompletedMessage - format cũ (MinIO file reference). Deprecated — giữ để backward compat.
// Sẽ xóa ở M5 Phase C sau khi analysis-srv hoàn toàn migrate sang format mới.
type LegacyBatchCompletedMessage struct {
	BatchID     string    `json:"batch_id"`
	ProjectID   string    `json:"project_id"`
	CampaignID  string    `json:"campaign_id,omitempty"`
	FileURL     string    `json:"file_url"`
	RecordCount int       `json:"record_count"`
	CompletedAt time.Time `json:"completed_at"`
}

// BatchCompletedMessage - format mới (direct payload). Analysis-srv phải produce format này.
type BatchCompletedMessage struct {
	ProjectID  string           `json:"project_id"`
	CampaignID string           `json:"campaign_id"`
	Documents  []InsightMessage `json:"documents"`
}

// InsightMessage - 1 document (POST, COMMENT, hoặc REPLY) trong documents[].
// Schema match 1:1 với contract.md Section 2.
type InsightMessage struct {
	Identity InsightIdentity `json:"identity"`
	Content  InsightContent  `json:"content"`
	NLP      InsightNLP      `json:"nlp"`
	Business InsightBusiness `json:"business"`
	RAG      bool            `json:"rag"`
}

type InsightIdentity struct {
	UapID        string `json:"uap_id"`         // primary key, Qdrant Point ID
	UapType      string `json:"uap_type"`        // "post" | "comment" | "reply"
	UapMediaType string `json:"uap_media_type"`  // "video" | "image" | "carousel" | "text" | "live" | "other"
	Platform     string `json:"platform"`        // "TIKTOK" | "FACEBOOK" | "INSTAGRAM" | "YOUTUBE" | "OTHER"
	PublishedAt  string `json:"published_at"`    // RFC3339 UTC
}

type InsightContent struct {
	CleanText string `json:"clean_text"` // TEXT ĐỂ EMBED
	Summary   string `json:"summary"`    // display snippet, không embed
}

type InsightNLP struct {
	Sentiment InsightSentiment `json:"sentiment"`
	Aspects   []InsightAspect  `json:"aspects"`
	Entities  []InsightEntity  `json:"entities"`
}

type InsightSentiment struct {
	Label string  `json:"label"` // "POSITIVE" | "NEGATIVE" | "NEUTRAL" | "MIXED"
	Score float64 `json:"score"` // -1.0 to 1.0
}

type InsightAspect struct {
	Aspect   string `json:"aspect"`   // e.g. "GENTLE", "TEXTURE"
	Polarity string `json:"polarity"` // "POSITIVE" | "NEGATIVE" | "NEUTRAL"
}

type InsightEntity struct {
	Type  string `json:"type"`  // "BRAND" | "PRODUCT" | "PERSON" | "LOCATION" | "OTHER"
	Value string `json:"value"` // e.g. "Cetaphil"
}

type InsightBusiness struct {
	Impact InsightImpact `json:"impact"`
}

type InsightImpact struct {
	Engagement  InsightEngagement `json:"engagement"`
	ImpactScore float64           `json:"impact_score"` // 0.0–100.0
	Priority    string            `json:"priority"`     // "LOW" | "MEDIUM" | "HIGH" | "CRITICAL"
}

type InsightEngagement struct {
	Likes    int `json:"likes"`
	Comments int `json:"comments"`
	Shares   int `json:"shares"`
	Views    int `json:"views"`
}

// ── Layer 2: analytics.insights.published ────────────────────────────────────

// InsightsPublishedMessage - 1 insight card. Schema match 1:1 với contract.md Section 3.
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

// ── Layer 1: analytics.report.digest ─────────────────────────────────────────

// ReportDigestMessage - 1 digest per run. Schema match 1:1 với contract.md Section 4.
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
	BuzzScoreProxy      *float64 `json:"buzz_score_proxy"`  // nullable
	QualityScore        *float64 `json:"quality_score"`     // nullable
	RepresentativeTexts []string `json:"representative_texts"`
}

type TopIssue struct {
	IssueCategory      string       `json:"issue_category"`
	MentionCount       int          `json:"mention_count"`
	IssuePressureProxy float64      `json:"issue_pressure_proxy"`
	SeverityMix        *SeverityMix `json:"severity_mix"` // nullable
}

type SeverityMix struct {
	Low    float64 `json:"low"`
	Medium float64 `json:"medium"`
	High   float64 `json:"high"`
}
```

---

### 2. `internal/indexing/types.go`

**APPEND** vào cuối file (không xóa code hiện tại). Thêm các input/output types mới:

```go
// ── Layer 2: IndexInsight ────────────────────────────────────────────────────

// IndexInsightInput - Input cho IndexInsight() usecase method (Layer 2)
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

// ── Layer 1: IndexDigest ─────────────────────────────────────────────────────

// IndexDigestInput - Input cho IndexDigest() usecase method (Layer 1)
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

// ── Layer 3: IndexBatch ──────────────────────────────────────────────────────

// IndexBatchInput - Input cho IndexBatch() usecase method (Layer 3, format mới).
// Thay thế IndexInput (vẫn giữ IndexInput cho backward compat với legacy MinIO flow).
type IndexBatchInput struct {
	ProjectID  string
	CampaignID string
	Documents  []InsightMessageInput
}

// InsightMessageInput - domain-level representation của 1 document.
// Mirror của kafka.InsightMessage nhưng là domain type (delivery → domain boundary).
type InsightMessageInput struct {
	Identity InsightIdentityInput
	Content  InsightContentInput
	NLP      InsightNLPInput
	Business InsightBusinessInput
	RAG      bool
}

type InsightIdentityInput struct {
	UapID        string
	UapType      string
	UapMediaType string
	Platform     string
	PublishedAt  string
}

type InsightContentInput struct {
	CleanText string
	Summary   string
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
	Impact InsightImpactInput
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
```

---

### 3. `internal/indexing/interface.go`

**REPLACE toàn bộ file**:

```go
package indexing

import (
	"context"
)

//go:generate mockery --name UseCase
type UseCase interface {
	// Layer 3 — legacy MinIO flow (deprecated, giữ backward compat)
	Index(ctx context.Context, input IndexInput) (IndexOutput, error)

	// Layer 3 — new direct payload flow
	IndexBatch(ctx context.Context, input IndexBatchInput) (IndexBatchOutput, error)

	// Layer 2 — insight cards
	IndexInsight(ctx context.Context, input IndexInsightInput) (IndexInsightOutput, error)

	// Layer 1 — report digest
	IndexDigest(ctx context.Context, input IndexDigestInput) (IndexDigestOutput, error)

	// Maintenance
	RetryFailed(ctx context.Context, ip RetryFailedInput) (RetryFailedOutput, error)
	Reconcile(ctx context.Context, ip ReconcileInput) (ReconcileOutput, error)
	GetStatistics(ctx context.Context, projectID string) (StatisticOutput, error)
}
```

---

### 4. `internal/indexing/errors.go`

**APPEND** 3 errors mới vào cuối block `var(...)`:

```go
package indexing

import "errors"

var (
	ErrAlreadyIndexed       = errors.New("indexing: record already indexed")
	ErrContentTooShort      = errors.New("indexing: content too short")
	ErrDuplicateContent     = errors.New("indexing: duplicate content")
	ErrEmbeddingFailed      = errors.New("indexing: embedding generation failed")
	ErrFileDownloadFailed   = errors.New("indexing: file download failed")
	ErrFileNotFound         = errors.New("indexing: file not found")
	ErrFileParseFailed      = errors.New("indexing: file parse failed")
	ErrInvalidAnalyticsData = errors.New("indexing: invalid analytics data")
	ErrQdrantUpsertFailed   = errors.New("indexing: qdrant upsert failed")
	ErrCountDocument        = errors.New("indexing: count document failed")

	// NEW — Layer 2 & 1
	ErrSkippedByGate       = errors.New("indexing: skipped by should_index gate")
	ErrInsightTitleEmpty   = errors.New("indexing: insight title is empty")
	ErrDigestBuildFailed   = errors.New("indexing: digest prose build failed")
	ErrInsightSummaryEmpty = errors.New("indexing: insight summary is empty")
)
```

---

## Verification Checklist

```bash
# 1. Build phải pass (interface chưa được implement đầy đủ sẽ lỗi ở usecase)
# → Sẽ lỗi compile vì UseCase interface mở rộng nhưng implUseCase chưa có IndexBatch/IndexInsight/IndexDigest
# → Đây là expected behavior ở M2 — sẽ fix ở M3/M4/M5 khi implement usecase methods
# Kiểm tra chỉ bằng cách type-check:
go vet ./internal/indexing/delivery/... ./internal/indexing/types.go

# 2. JSON round-trip test với sample data
# File: documents/refactor-input/outputSRM/insights/insights.jsonl
# Mỗi line là 1 InsightsPublishedMessage — unmarshal và kiểm tra fields

# 3. Verify JSON tags match contract.md
# Mở contract.md Section 3 và 4, so sánh từng field name với JSON tags trong InsightsPublishedMessage và ReportDigestMessage
```

## Notes

- `time` package import đã có sẵn trong `types.go` — không cần thêm
- `LegacyBatchCompletedMessage` ở `type.go`: giữ tên `BatchCompletedMessage` cũ trong struct name là không được vì conflict — đã rename sang `Legacy...`
- Mockery cần chạy lại sau khi thay đổi interface: `go generate ./internal/indexing/...`
