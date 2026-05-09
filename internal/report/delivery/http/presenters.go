package http

import (
	"encoding/json"

	"knowledge-srv/internal/report"
)

type generateReportReq struct {
	CampaignID            string         `json:"campaign_id" binding:"required"`
	ReportType            string         `json:"report_type" binding:"required"`
	Title                 string         `json:"title,omitempty"`
	Filters               *reportFilters `json:"filters,omitempty"`
	Sections              []string       `json:"sections,omitempty"`
	Prompt                string         `json:"prompt,omitempty"`
	Source                string         `json:"source,omitempty"`
	CompetitorURLs        []string       `json:"competitor_urls,omitempty"`
	MaxPostsPerCompetitor int            `json:"max_posts_per_competitor,omitempty"`
}

type reportFilters struct {
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

func (r generateReportReq) toInput() report.GenerateInput {
	input := report.GenerateInput{
		CampaignID: r.CampaignID,
		ReportType: r.ReportType,
		Title:      r.Title,
	}
	if r.Filters != nil {
		input.Filters = report.ReportFilters{
			Sentiments:            r.Filters.Sentiments,
			Aspects:               r.Filters.Aspects,
			Platforms:             r.Filters.Platforms,
			DateFrom:              r.Filters.DateFrom,
			DateTo:                r.Filters.DateTo,
			RiskLevels:            r.Filters.RiskLevels,
			Sections:              r.Filters.Sections,
			Prompt:                r.Filters.Prompt,
			Source:                r.Filters.Source,
			CompetitorURLs:        r.Filters.CompetitorURLs,
			MaxPostsPerCompetitor: r.Filters.MaxPostsPerCompetitor,
		}
	}
	if len(r.Sections) > 0 {
		input.Filters.Sections = r.Sections
	}
	if r.Prompt != "" {
		input.Filters.Prompt = r.Prompt
	}
	if r.Source != "" {
		input.Filters.Source = r.Source
	}
	if len(r.CompetitorURLs) > 0 {
		input.Filters.CompetitorURLs = r.CompetitorURLs
	}
	if r.MaxPostsPerCompetitor > 0 {
		input.Filters.MaxPostsPerCompetitor = r.MaxPostsPerCompetitor
	}
	return input
}

type listReportsReq struct {
	CampaignID string
	Status     string
	Page       int
	PageSize   int
}

func (r listReportsReq) toInput() report.ListReportsInput {
	return report.ListReportsInput{
		CampaignID: r.CampaignID,
		Status:     r.Status,
		Page:       r.Page,
		PageSize:   r.PageSize,
	}
}

type getReportReq struct {
	ReportID string
}

func (r getReportReq) toInput() report.GetReportInput {
	return report.GetReportInput{
		ReportID: r.ReportID,
	}
}

type getReportProcessReq struct {
	ReportID string
}

func (r getReportProcessReq) toInput() report.GetReportProcessInput {
	return report.GetReportProcessInput{
		ReportID: r.ReportID,
	}
}

type listReportPostsReq struct {
	ReportID  string
	Page      int
	PageSize  int
	Sentiment string
	Platform  string
}

func (r listReportPostsReq) toInput() report.ListReportPostsInput {
	return report.ListReportPostsInput{
		ReportID:  r.ReportID,
		Page:      r.Page,
		PageSize:  r.PageSize,
		Sentiment: r.Sentiment,
		Platform:  r.Platform,
	}
}

type listPostCommentsReq struct {
	PostID   string
	Page     int
	PageSize int
}

func (r listPostCommentsReq) toInput() report.ListPostCommentsInput {
	return report.ListPostCommentsInput{
		PostID:   r.PostID,
		Page:     r.Page,
		PageSize: r.PageSize,
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

type cancelReportReq struct {
	ReportID string
}

func (r cancelReportReq) toInput() report.CancelReportInput {
	return report.CancelReportInput{ReportID: r.ReportID}
}

type retryReportReq struct {
	ReportID string
}

func (r retryReportReq) toInput() report.RetryReportInput {
	return report.RetryReportInput{ReportID: r.ReportID}
}

type generateReportResp struct {
	ReportID string `json:"report_id"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

type listReportsResp struct {
	Items    []reportResp `json:"items"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
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

type reportProcessResp struct {
	ProcessID    string             `json:"process_id"`
	Status       string             `json:"status"`
	StartedAt    string             `json:"started_at"`
	FinishedAt   string             `json:"finished_at,omitempty"`
	Progress     reportProgressResp `json:"progress"`
	ErrorMessage string             `json:"error_message,omitempty"`
}

type reportProgressResp struct {
	Crawled       int                      `json:"crawled"`
	Target        int                      `json:"target"`
	PerCompetitor []competitorProgressResp `json:"per_competitor"`
}

type competitorProgressResp struct {
	URL      string `json:"url"`
	Platform string `json:"platform"`
	Crawled  int    `json:"crawled"`
	Target   int    `json:"target"`
	Status   string `json:"status"`
}

type listReportPostsResp struct {
	Items    []reportPostResp `json:"items"`
	Total    int              `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

type reportPostResp struct {
	ID                 string                   `json:"id"`
	ReportID           string                   `json:"report_id"`
	CompetitorURL      string                   `json:"competitor_url"`
	Platform           string                   `json:"platform"`
	Author             string                   `json:"author"`
	AuthorAvatar       string                   `json:"author_avatar,omitempty"`
	Content            string                   `json:"content"`
	PostedAt           string                   `json:"posted_at"`
	URL                string                   `json:"url"`
	Engagement         reportPostEngagementResp `json:"engagement"`
	Sentiment          string                   `json:"sentiment"`
	CommentCount       int                      `json:"comment_count"`
	SentimentBreakdown *sentimentBreakdownResp  `json:"sentiment_breakdown,omitempty"`
	TopKeywords        []string                 `json:"top_keywords,omitempty"`
}

type reportPostEngagementResp struct {
	Likes    int `json:"likes"`
	Comments int `json:"comments"`
	Shares   int `json:"shares"`
	Views    int `json:"views"`
}

type sentimentBreakdownResp struct {
	Positive int `json:"positive"`
	Neutral  int `json:"neutral"`
	Negative int `json:"negative"`
}

type listPostCommentsResp struct {
	Items    []reportCommentResp `json:"items"`
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

type reportCommentResp struct {
	ID        string              `json:"id"`
	PostID    string              `json:"post_id"`
	Author    string              `json:"author"`
	Content   string              `json:"content"`
	CreatedAt string              `json:"created_at"`
	Likes     int                 `json:"likes"`
	Sentiment string              `json:"sentiment"`
	Replies   []reportCommentResp `json:"replies,omitempty"`
}

type downloadResp struct {
	DownloadURL string `json:"download_url"`
	ExpiresAt   string `json:"expires_at"`
	FileName    string `json:"file_name"`
	FileSize    int64  `json:"file_size"`
}

type cancelResp struct {
	OK bool `json:"ok"`
}

type retryResp struct {
	ReportID  string `json:"report_id"`
	ProcessID string `json:"process_id"`
	Status    string `json:"status"`
}

func (h *handler) newGenerateReportResp(o report.GenerateOutput) generateReportResp {
	return generateReportResp{
		ReportID: o.ReportID,
		Status:   o.Status,
		Message:  o.Message,
	}
}

func (h *handler) newListReportsResp(o report.ListReportsOutput) listReportsResp {
	items := make([]reportResp, 0, len(o.Items))
	for _, item := range o.Items {
		items = append(items, h.newReportResp(item))
	}
	return listReportsResp{
		Items:    items,
		Total:    o.Total,
		Page:     o.Page,
		PageSize: o.PageSize,
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

func (h *handler) newReportProcessResp(o report.ReportProcessOutput) reportProcessResp {
	perCompetitor := make([]competitorProgressResp, 0, len(o.Progress.PerCompetitor))
	for _, item := range o.Progress.PerCompetitor {
		perCompetitor = append(perCompetitor, competitorProgressResp{
			URL:      item.URL,
			Platform: item.Platform,
			Crawled:  item.Crawled,
			Target:   item.Target,
			Status:   item.Status,
		})
	}
	return reportProcessResp{
		ProcessID:  o.ProcessID,
		Status:     o.Status,
		StartedAt:  o.StartedAt,
		FinishedAt: o.FinishedAt,
		Progress: reportProgressResp{
			Crawled:       o.Progress.Crawled,
			Target:        o.Progress.Target,
			PerCompetitor: perCompetitor,
		},
		ErrorMessage: o.ErrorMessage,
	}
}

func (h *handler) newListReportPostsResp(o report.ListReportPostsOutput) listReportPostsResp {
	items := make([]reportPostResp, 0, len(o.Items))
	for _, item := range o.Items {
		items = append(items, h.newReportPostResp(item))
	}
	return listReportPostsResp{
		Items:    items,
		Total:    o.Total,
		Page:     o.Page,
		PageSize: o.PageSize,
	}
}

func (h *handler) newReportPostResp(o report.ReportPostOutput) reportPostResp {
	var breakdown *sentimentBreakdownResp
	if o.SentimentBreakdown != nil {
		breakdown = &sentimentBreakdownResp{
			Positive: o.SentimentBreakdown.Positive,
			Neutral:  o.SentimentBreakdown.Neutral,
			Negative: o.SentimentBreakdown.Negative,
		}
	}
	return reportPostResp{
		ID:            o.ID,
		ReportID:      o.ReportID,
		CompetitorURL: o.CompetitorURL,
		Platform:      o.Platform,
		Author:        o.Author,
		AuthorAvatar:  o.AuthorAvatar,
		Content:       o.Content,
		PostedAt:      o.PostedAt,
		URL:           o.URL,
		Engagement: reportPostEngagementResp{
			Likes:    o.Engagement.Likes,
			Comments: o.Engagement.Comments,
			Shares:   o.Engagement.Shares,
			Views:    o.Engagement.Views,
		},
		Sentiment:          o.Sentiment,
		CommentCount:       o.CommentCount,
		SentimentBreakdown: breakdown,
		TopKeywords:        o.TopKeywords,
	}
}

func (h *handler) newListPostCommentsResp(o report.ListPostCommentsOutput) listPostCommentsResp {
	items := make([]reportCommentResp, 0, len(o.Items))
	for _, item := range o.Items {
		items = append(items, h.newReportCommentResp(item))
	}
	return listPostCommentsResp{
		Items:    items,
		Total:    o.Total,
		Page:     o.Page,
		PageSize: o.PageSize,
	}
}

func (h *handler) newReportCommentResp(o report.ReportCommentOutput) reportCommentResp {
	replies := make([]reportCommentResp, 0, len(o.Replies))
	for _, reply := range o.Replies {
		replies = append(replies, h.newReportCommentResp(reply))
	}
	return reportCommentResp{
		ID:        o.ID,
		PostID:    o.PostID,
		Author:    o.Author,
		Content:   o.Content,
		CreatedAt: o.CreatedAt,
		Likes:     o.Likes,
		Sentiment: o.Sentiment,
		Replies:   replies,
	}
}

func (h *handler) newDownloadResp(o report.DownloadOutput) downloadResp {
	return downloadResp{
		DownloadURL: o.DownloadURL,
		ExpiresAt:   o.ExpiresAt,
		FileName:    o.FileName,
		FileSize:    o.FileSize,
	}
}

func (h *handler) newCancelResp(o report.CancelOutput) cancelResp {
	return cancelResp{OK: o.OK}
}

func (h *handler) newRetryResp(o report.RetryOutput) retryResp {
	return retryResp{
		ReportID:  o.ReportID,
		ProcessID: o.ProcessID,
		Status:    o.Status,
	}
}
