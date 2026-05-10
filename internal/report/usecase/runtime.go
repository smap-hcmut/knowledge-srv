package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/report"
	"knowledge-srv/internal/report/repository"
	"knowledge-srv/internal/search"
	"strings"
	"time"
)

const (
	defaultListPageSize  = 20
	maxListPageSize      = 50
	reportSearchMinScore = 0.45
)

func (uc *implUseCase) ListReports(ctx context.Context, sc model.Scope, input report.ListReportsInput) (report.ListReportsOutput, error) {
	if input.CampaignID == "" {
		return report.ListReportsOutput{}, report.ErrCampaignRequired
	}

	page, pageSize, offset := normalizePagination(input.Page, input.PageSize)
	opts := repository.ListReportsOptions{
		CampaignID: input.CampaignID,
		UserID:     sc.UserID,
		Status:     normalizeStatus(input.Status),
		Limit:      pageSize,
		Offset:     offset,
	}

	total, err := uc.repo.CountReports(ctx, opts)
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.ListReports: Failed to count reports: %v", err)
		return report.ListReportsOutput{}, report.ErrGenerationFailed
	}

	reports, err := uc.repo.ListReports(ctx, opts)
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.ListReports: Failed to list reports: %v", err)
		return report.ListReportsOutput{}, report.ErrGenerationFailed
	}

	items := make([]report.ReportOutput, 0, len(reports))
	for _, rpt := range reports {
		items = append(items, uc.buildReportOutput(rpt))
	}

	return report.ListReportsOutput{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (uc *implUseCase) GetReportProcess(ctx context.Context, sc model.Scope, input report.GetReportProcessInput) (report.ReportProcessOutput, error) {
	rpt, err := uc.repo.GetReportByID(ctx, input.ReportID)
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.GetReportProcess: Failed to get report: %v", err)
		return report.ReportProcessOutput{}, report.ErrReportNotFound
	}
	if !canAccessReport(sc, rpt) {
		return report.ReportProcessOutput{}, report.ErrReportForbidden
	}
	return uc.buildProcessOutput(rpt), nil
}

func (uc *implUseCase) ListReportPosts(ctx context.Context, sc model.Scope, input report.ListReportPostsInput) (report.ListReportPostsOutput, error) {
	rpt, err := uc.repo.GetReportByID(ctx, input.ReportID)
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.ListReportPosts: Failed to get report: %v", err)
		return report.ListReportPostsOutput{}, report.ErrReportNotFound
	}
	if !canAccessReport(sc, rpt) {
		return report.ListReportPostsOutput{}, report.ErrReportForbidden
	}

	page, pageSize, offset := normalizePagination(input.Page, input.PageSize)
	filters := decodeReportFilters(rpt.Filters)
	if input.Sentiment != "" {
		filters.Sentiments = []string{strings.ToUpper(input.Sentiment)}
	}
	if input.Platform != "" {
		filters.Platforms = []string{strings.ToLower(input.Platform)}
	}

	searchOutput, err := uc.searchUC.Search(ctx, sc, search.SearchInput{
		CampaignID: rpt.CampaignID,
		Query:      buildReportEvidenceQuery(rpt.ReportType, filters),
		Limit:      maxListPageSize,
		MinScore:   reportSearchMinScore,
		Filters: search.SearchFilters{
			Sentiments: filters.Sentiments,
			Aspects:    filters.Aspects,
			Platforms:  filters.Platforms,
			DateFrom:   filters.DateFrom,
			DateTo:     filters.DateTo,
			RiskLevels: filters.RiskLevels,
		},
	})
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.ListReportPosts: Search failed: %v", err)
		return report.ListReportPostsOutput{}, report.ErrGenerationFailed
	}
	searchOutput = sanitizeReportSearchOutput(searchOutput)

	results := searchOutput.Results
	if offset > len(results) {
		results = nil
	} else {
		results = results[offset:]
		if len(results) > pageSize {
			results = results[:pageSize]
		}
	}

	items := make([]report.ReportPostOutput, 0, len(results))
	for _, result := range results {
		items = append(items, mapSearchResultToReportPost(rpt.ID, result))
	}

	return report.ListReportPostsOutput{
		Items:    items,
		Total:    searchOutput.TotalFound,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (uc *implUseCase) ListPostComments(ctx context.Context, sc model.Scope, input report.ListPostCommentsInput) (report.ListPostCommentsOutput, error) {
	page, pageSize, _ := normalizePagination(input.Page, input.PageSize)
	return report.ListPostCommentsOutput{
		Items:    []report.ReportCommentOutput{},
		Total:    0,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (uc *implUseCase) CancelReport(ctx context.Context, sc model.Scope, input report.CancelReportInput) (report.CancelOutput, error) {
	rpt, err := uc.repo.GetReportByID(ctx, input.ReportID)
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.CancelReport: Failed to get report: %v", err)
		return report.CancelOutput{}, report.ErrReportNotFound
	}
	if !canAccessReport(sc, rpt) {
		return report.CancelOutput{}, report.ErrReportForbidden
	}
	if rpt.Status == report.StatusCompleted || rpt.Status == report.StatusFailed || rpt.Status == report.StatusCancelled {
		return report.CancelOutput{OK: true}, nil
	}
	if err := uc.repo.UpdateCancelled(ctx, repository.UpdateCancelledOptions{ReportID: input.ReportID}); err != nil {
		uc.l.Errorf(ctx, "report.usecase.CancelReport: Failed to cancel report: %v", err)
		return report.CancelOutput{}, report.ErrGenerationFailed
	}
	return report.CancelOutput{OK: true}, nil
}

func (uc *implUseCase) RetryReport(ctx context.Context, sc model.Scope, input report.RetryReportInput) (report.RetryOutput, error) {
	rpt, err := uc.repo.GetReportByID(ctx, input.ReportID)
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.RetryReport: Failed to get report: %v", err)
		return report.RetryOutput{}, report.ErrReportNotFound
	}
	if !canAccessReport(sc, rpt) {
		return report.RetryOutput{}, report.ErrReportForbidden
	}
	if rpt.Status == report.StatusProcessing {
		return report.RetryOutput{ReportID: rpt.ID, ProcessID: rpt.ID, Status: report.StatusProcessing}, nil
	}

	filters := decodeReportFilters(rpt.Filters)
	if err := uc.repo.UpdateProcessing(ctx, repository.UpdateProcessingOptions{ReportID: rpt.ID}); err != nil {
		uc.l.Errorf(ctx, "report.usecase.RetryReport: Failed to set processing: %v", err)
		return report.RetryOutput{}, report.ErrGenerationFailed
	}

	inputForGeneration := report.GenerateInput{
		CampaignID: rpt.CampaignID,
		ReportType: rpt.ReportType,
		Title:      rpt.Title,
		Filters:    filters,
	}
	go func() {
		uc.reportSem <- struct{}{}
		defer func() { <-uc.reportSem }()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		uc.generateInBackground(ctx, rpt.ID, inputForGeneration)
	}()

	return report.RetryOutput{ReportID: rpt.ID, ProcessID: rpt.ID, Status: report.StatusProcessing}, nil
}

func normalizePagination(page, pageSize int) (int, int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultListPageSize
	}
	if pageSize > maxListPageSize {
		pageSize = maxListPageSize
	}
	return page, pageSize, (page - 1) * pageSize
}

func normalizeStatus(status string) string {
	status = strings.ToUpper(strings.TrimSpace(status))
	switch status {
	case report.StatusProcessing, report.StatusCompleted, report.StatusFailed, report.StatusCancelled:
		return status
	default:
		return ""
	}
}

func canAccessReport(sc model.Scope, rpt *model.Report) bool {
	if rpt == nil {
		return false
	}
	return sc.IsAdmin() || (sc.UserID != "" && sc.UserID == rpt.UserID)
}

func decodeReportFilters(raw json.RawMessage) report.ReportFilters {
	if len(raw) == 0 || string(raw) == "null" {
		return report.ReportFilters{}
	}
	var filters report.ReportFilters
	if err := json.Unmarshal(raw, &filters); err != nil {
		return report.ReportFilters{}
	}
	return filters
}

func (uc *implUseCase) buildProcessOutput(rpt *model.Report) report.ReportProcessOutput {
	status := processStatusFromReportStatus(rpt.Status)
	target := uc.config.MaxDocs
	crawled := 0
	if rpt.TotalDocsAnalyzed > 0 {
		target = rpt.TotalDocsAnalyzed
	}
	if rpt.Status == report.StatusCompleted {
		crawled = target
	}

	progress := report.ReportProcessProgress{
		Crawled: crawled,
		Target:  target,
	}

	filters := decodeReportFilters(rpt.Filters)
	for _, url := range filters.CompetitorURLs {
		progress.PerCompetitor = append(progress.PerCompetitor, report.ReportCompetitorProgress{
			URL:      url,
			Platform: platformFromURL(url),
			Crawled:  crawled,
			Target:   target,
			Status:   status,
		})
	}

	out := report.ReportProcessOutput{
		ProcessID:    rpt.ID,
		Status:       status,
		StartedAt:    rpt.CreatedAt.Format(time.RFC3339),
		Progress:     progress,
		ErrorMessage: rpt.ErrorMessage,
	}
	if rpt.CompletedAt != nil {
		out.FinishedAt = rpt.CompletedAt.Format(time.RFC3339)
	}
	return out
}

func processStatusFromReportStatus(status string) string {
	switch status {
	case report.StatusCompleted:
		return "done"
	case report.StatusFailed:
		return "failed"
	case report.StatusCancelled:
		return "cancelled"
	default:
		return "running"
	}
}

func buildReportEvidenceQuery(reportType string, filters report.ReportFilters) string {
	return buildReportRetrievalQuery(reportType, filters)
}

func mapSearchResultToReportPost(reportID string, result search.SearchResult) report.ReportPostOutput {
	metadata := result.Metadata
	engagement := report.ReportPostEngagement{
		Likes:    intFromPayload(metadata, "likes"),
		Comments: intFromPayload(metadata, "comments"),
		Shares:   intFromPayload(metadata, "shares"),
		Views:    intFromPayload(metadata, "views"),
	}
	if engagement.Likes == 0 && engagement.Comments == 0 && engagement.Shares == 0 && engagement.Views == 0 {
		engagement = report.ReportPostEngagement{
			Likes:    intFromNestedPayload(metadata, "metadata", "engagement", "likes"),
			Comments: intFromNestedPayload(metadata, "metadata", "engagement", "comments"),
			Shares:   intFromNestedPayload(metadata, "metadata", "engagement", "shares"),
			Views:    intFromNestedPayload(metadata, "metadata", "engagement", "views"),
		}
	}

	author := authorFromMetadata(metadata)
	authorAvatar := firstURL(
		stringFromNestedPayload(metadata, "metadata", "author_avatar"),
		stringFromNestedPayload(metadata, "metadata", "author_avatar_url"),
		stringFromPayload(metadata, "author_avatar"),
		stringFromPayload(metadata, "author_avatar_url"),
	)
	url := sourceURLFromMetadata(metadata)
	postedAt := time.Now().Format(time.RFC3339)
	if result.ContentCreatedAt > 0 {
		postedAt = time.Unix(result.ContentCreatedAt, 0).Format(time.RFC3339)
	} else if raw := stringFromPayload(metadata, "published_at"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			postedAt = parsed.Format(time.RFC3339)
		}
	}

	return report.ReportPostOutput{
		ID:            firstNonEmpty(result.ID, fmt.Sprintf("%s:%d", reportID, result.ContentCreatedAt)),
		ReportID:      reportID,
		CompetitorURL: "",
		Platform:      strings.ToLower(result.Platform),
		Author:        author,
		AuthorAvatar:  authorAvatar,
		Content:       result.Content,
		PostedAt:      postedAt,
		URL:           url,
		Engagement:    engagement,
		Sentiment:     normalizeSentiment(result.OverallSentiment),
		CommentCount:  engagement.Comments,
		TopKeywords:   result.Keywords,
	}
}

func normalizeSentiment(sentiment string) string {
	switch strings.ToUpper(strings.TrimSpace(sentiment)) {
	case "POSITIVE":
		return "positive"
	case "NEGATIVE":
		return "negative"
	default:
		return "neutral"
	}
}

func intFromPayload(payload map[string]interface{}, key string) int {
	if payload == nil {
		return 0
	}
	switch v := payload[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	default:
		return 0
	}
}

func stringFromPayload(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func stringFromNestedPayload(payload map[string]interface{}, parentKey, childKey string) string {
	if payload == nil {
		return ""
	}
	parent, ok := payload[parentKey]
	if !ok {
		return ""
	}
	obj, ok := parent.(map[string]interface{})
	if !ok {
		return ""
	}
	value, ok := obj[childKey]
	if !ok {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstURL(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		lower := strings.ToLower(value)
		if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
			return value
		}
	}
	return ""
}

func intFromNestedPayload(payload map[string]interface{}, keys ...string) int {
	var current interface{} = payload
	for _, key := range keys {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return 0
		}
		current = obj[key]
	}
	switch v := current.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	default:
		return 0
	}
}

func platformFromURL(url string) string {
	lower := strings.ToLower(url)
	switch {
	case strings.Contains(lower, "tiktok"):
		return "tiktok"
	case strings.Contains(lower, "youtube") || strings.Contains(lower, "youtu.be"):
		return "youtube"
	default:
		return "facebook"
	}
}
