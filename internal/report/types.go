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
)

type GenerateInput struct {
	CampaignID string
	ReportType string
	Filters    ReportFilters
	Title      string
}

type ReportFilters struct {
	Sentiments []string `json:"sentiments,omitempty"`
	Aspects    []string `json:"aspects,omitempty"`
	Platforms  []string `json:"platforms,omitempty"`
	DateFrom   *int64   `json:"date_from,omitempty"`
	DateTo     *int64   `json:"date_to,omitempty"`
	RiskLevels []string `json:"risk_levels,omitempty"`
}

func (f ReportFilters) ToJSON() ([]byte, error) {
	return json.Marshal(f)
}

type GetReportInput struct {
	ReportID string
}

type DownloadReportInput struct {
	ReportID string
}

type GenerateOutput struct {
	ReportID string `json:"report_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
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

type DownloadOutput struct {
	DownloadURL string `json:"download_url"`
	ExpiresAt   string `json:"expires_at"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
}

type SectionTemplate struct {
	Title  string
	Prompt string
}
