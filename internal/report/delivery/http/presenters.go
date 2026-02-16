package http

import (
	"encoding/json"

	"knowledge-srv/internal/report"
)

type generateReportReq struct {
	CampaignID string         `json:"campaign_id" binding:"required"`
	ReportType string         `json:"report_type" binding:"required"`
	Title      string         `json:"title,omitempty"`
	Filters    *reportFilters `json:"filters,omitempty"`
}

type reportFilters struct {
	Sentiments []string `json:"sentiments,omitempty"`
	Aspects    []string `json:"aspects,omitempty"`
	Platforms  []string `json:"platforms,omitempty"`
	DateFrom   *int64   `json:"date_from,omitempty"`
	DateTo     *int64   `json:"date_to,omitempty"`
	RiskLevels []string `json:"risk_levels,omitempty"`
}

func (r generateReportReq) toInput() report.GenerateInput {
	input := report.GenerateInput{
		CampaignID: r.CampaignID,
		ReportType: r.ReportType,
		Title:      r.Title,
	}
	if r.Filters != nil {
		input.Filters = report.ReportFilters{
			Sentiments: r.Filters.Sentiments,
			Aspects:    r.Filters.Aspects,
			Platforms:  r.Filters.Platforms,
			DateFrom:   r.Filters.DateFrom,
			DateTo:     r.Filters.DateTo,
			RiskLevels: r.Filters.RiskLevels,
		}
	}
	return input
}

type getReportReq struct {
	ReportID string
}

func (r getReportReq) toInput() report.GetReportInput {
	return report.GetReportInput{
		ReportID: r.ReportID,
	}
}

type downloadReportReq struct {
	ReportID string
}

func (r downloadReportReq) toInput() report.DownloadReportInput {
	return report.DownloadReportInput{
		ReportID: r.ReportID,
	}
}

type generateReportResp struct {
	ReportID string `json:"report_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

type reportResp struct {
	ID                string      `json:"id"`
	CampaignID        string      `json:"campaign_id"`
	UserID            string      `json:"user_id"`
	Title             string      `json:"title"`
	ReportType        string      `json:"report_type"`
	Status            string      `json:"status"`
	ErrorMessage      string      `json:"error_message,omitempty"`
	FileFormat        string      `json:"file_format,omitempty"`
	FileSizeBytes     int64       `json:"file_size_bytes,omitempty"`
	TotalDocsAnalyzed int         `json:"total_docs_analyzed,omitempty"`
	SectionsCount     int         `json:"sections_count,omitempty"`
	GenerationTimeMs  int64       `json:"generation_time_ms,omitempty"`
	Filters           interface{} `json:"filters,omitempty" swaggertype:"object"`
	CompletedAt       *string     `json:"completed_at,omitempty"`
	CreatedAt         string      `json:"created_at"`
}

type downloadResp struct {
	DownloadURL string `json:"download_url"`
	ExpiresAt   string `json:"expires_at"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
}

func (h *handler) newGenerateReportResp(o report.GenerateOutput) generateReportResp {
	return generateReportResp{
		ReportID: o.ReportID,
		Status:   o.Status,
		Message:  o.Message,
	}
}

func (h *handler) newReportResp(o report.ReportOutput) reportResp {
	resp := reportResp{
		ID:                o.ID,
		CampaignID:        o.CampaignID,
		UserID:            o.UserID,
		Title:             o.Title,
		ReportType:        o.ReportType,
		Status:            o.Status,
		ErrorMessage:      o.ErrorMessage,
		FileFormat:        o.FileFormat,
		FileSizeBytes:     o.FileSizeBytes,
		TotalDocsAnalyzed: o.TotalDocsAnalyzed,
		SectionsCount:     o.SectionsCount,
		GenerationTimeMs:  o.GenerationTimeMs,
		CompletedAt:       o.CompletedAt,
		CreatedAt:         o.CreatedAt,
	}
	// Convert json.RawMessage to interface{} for Swagger compatibility
	if len(o.Filters) > 0 {
		var filters interface{}
		if err := json.Unmarshal(o.Filters, &filters); err == nil {
			resp.Filters = filters
		}
	}
	return resp
}

func (h *handler) newDownloadResp(o report.DownloadOutput) downloadResp {
	return downloadResp{
		DownloadURL: o.DownloadURL,
		ExpiresAt:   o.ExpiresAt,
		FileName:    o.FileName,
		FileSize:    o.FileSize,
	}
}
