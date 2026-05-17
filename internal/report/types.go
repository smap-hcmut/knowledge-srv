package report

import "encoding/json"

const (
	ReportTypeSummary    = "SUMMARY"
	ReportTypeComparison = "COMPARISON"
	ReportTypeTrend      = "TREND"
	ReportTypeAspectDeep = "ASPECT_DEEP_DIVE"
	StatusProcessing     = "PROCESSING"
	StatusCompleted      = "COMPLETED"
	StatusFailed         = "FAILED"
	StatusCancelled      = "CANCELLED"
)

type GenerateInput struct {
	CampaignID string
	ReportType string
	Filters    ReportFilters
	Title      string
}

type ReportFilters struct {
	Sentiments            []string `json:"sentiments,omitempty"`
	Aspects               []string `json:"aspects,omitempty"`
	Platforms             []string `json:"platforms,omitempty"`
	DateFrom              *int64   `json:"date_from,omitempty"`
	DateTo                *int64   `json:"date_to,omitempty"`
	RiskLevels            []string `json:"risk_levels,omitempty"`
	Sections              []string `json:"sections,omitempty"`
	Prompt                string   `json:"prompt,omitempty"`
	Source                string   `json:"source,omitempty"`
	CompetitorURLs        []string `json:"competitor_urls,omitempty"`
	MaxPostsPerCompetitor int      `json:"max_posts_per_competitor,omitempty"`
}

func (f ReportFilters) ToJSON() ([]byte, error) {
	return json.Marshal(f)
}

type GetReportInput struct {
	ReportID string
}

type ListReportsInput struct {
	CampaignID string
	Status     string
	Page       int
	PageSize   int
}

type DownloadReportInput struct {
	ReportID string
}

type GetReportContentInput struct {
	ReportID string
}

type GetReportProcessInput struct {
	ReportID string
}

type ListReportPostsInput struct {
	ReportID  string
	Page      int
	PageSize  int
	Sentiment string
	Platform  string
}

type ListPostCommentsInput struct {
	PostID   string
	Page     int
	PageSize int
}

type CancelReportInput struct {
	ReportID string
}

type RetryReportInput struct {
	ReportID string
}

type DeleteReportInput struct {
	ReportID string
}

type GenerateOutput struct {
	ReportID string `json:"report_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

type ListReportsOutput struct {
	Items    []ReportOutput `json:"items"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

type ReportOutput struct {
	ID                string          `json:"id"`
	CampaignID        string          `json:"campaign_id"`
	UserID            string          `json:"user_id"`
	Title             string          `json:"title"`
	ReportType        string          `json:"report_type"`
	Status            string          `json:"status"`
	ErrorMessage      string          `json:"error_message,omitempty"`
	FileFormat        string          `json:"file_format,omitempty"`
	FileSizeBytes     int64           `json:"file_size_bytes,omitempty"`
	TotalDocsAnalyzed int             `json:"total_docs_analyzed,omitempty"`
	SectionsCount     int             `json:"sections_count,omitempty"`
	GenerationTimeMs  int64           `json:"generation_time_ms,omitempty"`
	Filters           json.RawMessage `json:"filters,omitempty"`
	CompletedAt       *string         `json:"completed_at,omitempty"`
	CreatedAt         string          `json:"created_at"`
}

type ReportProcessOutput struct {
	ProcessID    string                `json:"process_id"`
	Status       string                `json:"status"`
	StartedAt    string                `json:"started_at"`
	FinishedAt   string                `json:"finished_at,omitempty"`
	Progress     ReportProcessProgress `json:"progress"`
	ErrorMessage string                `json:"error_message,omitempty"`
}

type ReportProcessProgress struct {
	Crawled       int                        `json:"crawled"`
	Target        int                        `json:"target"`
	PerCompetitor []ReportCompetitorProgress `json:"per_competitor"`
}

type ReportCompetitorProgress struct {
	URL      string `json:"url"`
	Platform string `json:"platform"`
	Crawled  int    `json:"crawled"`
	Target   int    `json:"target"`
	Status   string `json:"status"`
}

type ListReportPostsOutput struct {
	Items    []ReportPostOutput `json:"items"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

type ReportPostOutput struct {
	ID                 string                    `json:"id"`
	ReportID           string                    `json:"report_id"`
	CompetitorURL      string                    `json:"competitor_url"`
	Platform           string                    `json:"platform"`
	Author             string                    `json:"author"`
	AuthorAvatar       string                    `json:"author_avatar,omitempty"`
	Content            string                    `json:"content"`
	PostedAt           string                    `json:"posted_at"`
	URL                string                    `json:"url"`
	Engagement         ReportPostEngagement      `json:"engagement"`
	Sentiment          string                    `json:"sentiment"`
	CommentCount       int                       `json:"comment_count"`
	SentimentBreakdown *ReportSentimentBreakdown `json:"sentiment_breakdown,omitempty"`
	TopKeywords        []string                  `json:"top_keywords,omitempty"`
}

type ReportPostEngagement struct {
	Likes    int `json:"likes"`
	Comments int `json:"comments"`
	Shares   int `json:"shares"`
	Views    int `json:"views"`
}

type ReportSentimentBreakdown struct {
	Positive int `json:"positive"`
	Neutral  int `json:"neutral"`
	Negative int `json:"negative"`
}

type ListPostCommentsOutput struct {
	Items    []ReportCommentOutput `json:"items"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

type ReportCommentOutput struct {
	ID        string                `json:"id"`
	PostID    string                `json:"post_id"`
	Author    string                `json:"author"`
	Content   string                `json:"content"`
	CreatedAt string                `json:"created_at"`
	Likes     int                   `json:"likes"`
	Sentiment string                `json:"sentiment"`
	Replies   []ReportCommentOutput `json:"replies,omitempty"`
}

type DownloadOutput struct {
	DownloadURL string `json:"download_url"`
	ExpiresAt   string `json:"expires_at"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
}

type ReportContentOutput struct {
	ReportID    string `json:"report_id"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
}

type CancelOutput struct {
	OK bool `json:"ok"`
}

type RetryOutput struct {
	ReportID  string `json:"report_id"`
	ProcessID string `json:"process_id"`
	Status    string `json:"status"`
}

type DeleteOutput struct {
	OK bool `json:"ok"`
}

type SectionTemplate struct {
	Title  string
	Prompt string
}
