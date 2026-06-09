package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/report"
	"knowledge-srv/internal/report/repository"
	"knowledge-srv/internal/search"
	"strings"
	"time"

	"github.com/smap-hcmut/shared-libs/go/minio"
)

// errNoDocsForReport marks the only permanent (non-retriable) failure mode:
// the campaign has no indexed evidence matching the filters. Retrying buys
// nothing — the user has to broaden filters or wait for more ingestion.
var errNoDocsForReport = errors.New("no relevant documents found for report generation")

// reportRetryAttempts caps how many times generateInBackground will re-run
// the pipeline on transient failures (LLM timeout, search/MinIO blip).
const (
	reportRetryAttempts = 2
	reportRetryBackoff  = 5 * time.Second
)

// generateInBackground runs the report generation pipeline with retry.
// Transient failures (LLM/search/upload) get one more attempt after a short
// backoff; the permanent "no docs" outcome short-circuits immediately.
func (uc *implUseCase) generateInBackground(ctx context.Context, reportID string, input report.GenerateInput) {
	// Panic recovery wraps the whole retry loop so a panic in attempt N still
	// records the failure on the report row instead of leaking up the goroutine.
	defer func() {
		if r := recover(); r != nil {
			uc.l.Errorf(ctx, "report.usecase.generateInBackground: panic recovered: %v", r)
			_ = uc.repo.UpdateFailed(ctx, repository.UpdateFailedOptions{
				ReportID:     reportID,
				ErrorMessage: fmt.Sprintf("internal panic: %v", r),
			})
		}
	}()

	var lastErr error
	for attempt := 1; attempt <= reportRetryAttempts; attempt++ {
		// If the user cancelled mid-retry, stop instead of overwriting the
		// cancelled status with another attempt's failure.
		if latest, err := uc.repo.GetReportByID(ctx, reportID); err == nil && latest.Status == report.StatusCancelled {
			uc.l.Infof(ctx, "report.usecase.generateInBackground: Report %s cancelled before attempt %d", reportID, attempt)
			return
		}

		uc.l.Infof(ctx, "report.usecase.generateInBackground: attempt %d/%d for report %s", attempt, reportRetryAttempts, reportID)
		err := uc.tryGenerate(ctx, reportID, input)
		if err == nil {
			return
		}
		lastErr = err
		if errors.Is(err, errNoDocsForReport) {
			// Permanent — broader filters or fresher ingestion required.
			_ = uc.repo.UpdateFailed(ctx, repository.UpdateFailedOptions{
				ReportID:     reportID,
				ErrorMessage: err.Error(),
			})
			return
		}
		if attempt < reportRetryAttempts {
			uc.l.Warnf(ctx, "report.usecase.generateInBackground: attempt %d failed for %s: %v — retrying in %s", attempt, reportID, err, reportRetryBackoff)
			select {
			case <-ctx.Done():
				_ = uc.repo.UpdateFailed(ctx, repository.UpdateFailedOptions{
					ReportID:     reportID,
					ErrorMessage: fmt.Sprintf("cancelled while retrying: %v", err),
				})
				return
			case <-time.After(reportRetryBackoff):
			}
		}
	}

	// All retries exhausted — record the last error so the UI can surface it.
	_ = uc.repo.UpdateFailed(ctx, repository.UpdateFailedOptions{
		ReportID:     reportID,
		ErrorMessage: fmt.Sprintf("retries exhausted (%d attempts): %v", reportRetryAttempts, lastErr),
	})
}

// tryGenerate runs a single attempt of the report pipeline. Returns
// errNoDocsForReport for the permanent "no evidence" outcome, or a raw
// error for retriable failures. nil on success.
func (uc *implUseCase) tryGenerate(ctx context.Context, reportID string, input report.GenerateInput) error {
	startTime := time.Now()

	// Phase 1: Aggregate - Search for relevant documents
	searchOutput, err := uc.aggregateDocs(ctx, input)
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.tryGenerate: Aggregate phase failed: %v", err)
		return fmt.Errorf("aggregate failed: %w", err)
	}

	if len(searchOutput.Results) == 0 {
		return errNoDocsForReport
	}

	totalDocs := len(searchOutput.Results)
	uc.l.Infof(ctx, "report.usecase.tryGenerate: Found %d documents for report %s", totalDocs, reportID)

	// Phase 2: Evidence - Select representative, business-grade documents.
	// Use the full retrieval set here so the evidence pack can cover sentiment
	// and platform diversity instead of only mirroring the top semantic hits.
	evidence := buildBusinessEvidencePack(searchOutput.Results, uc.config.SampleSize)
	analyticsSummary := uc.loadReportAnalyticsSummary(ctx, input.CampaignID)

	// Phase 3: Generate - one coherent business report, grounded by evidence IDs.
	prompt := buildBusinessReportPrompt(input, businessPromptData{
		TotalDocs:        totalDocs,
		Aggregation:      formatAggregation(searchOutput.Aggregations),
		AnalyticsSummary: analyticsSummary,
		Evidence:         formatBusinessEvidenceForPrompt(evidence),
		Sections:         strings.Join(input.Filters.Sections, ", "),
		CompetitorURLs:   strings.Join(input.Filters.CompetitorURLs, ", "),
	})

	llmCtx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	content, err := uc.llm.Generate(llmCtx, prompt)
	cancel()
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.tryGenerate: LLM generation failed: %v", err)
		return fmt.Errorf("LLM generation failed: %w", err)
	}
	content = normalizeBusinessReportMarkdown(content)
	sectionsCount := countBusinessSections(content)

	uc.l.Infof(ctx, "report.usecase.tryGenerate: Generated business report with %d sections for report %s", sectionsCount, reportID)

	// Phase 4: Compile - Assemble markdown and upload
	markdown := compileBusinessMarkdown(input, content, evidence, totalDocs)

	objectName := fmt.Sprintf("reports/%s.md", reportID)
	fileBytes := []byte(markdown)

	if err := uc.ensureReportBucket(ctx); err != nil {
		uc.l.Errorf(ctx, "report.usecase.tryGenerate: Storage setup failed for bucket %q: %v", uc.config.ReportBucket, err)
		return fmt.Errorf("storage setup failed: %w", err)
	}

	if _, err := uc.minio.UploadFile(ctx, &minio.UploadRequest{
		BucketName:  uc.config.ReportBucket,
		ObjectName:  objectName,
		Reader:      bytes.NewReader(fileBytes),
		Size:        int64(len(fileBytes)),
		ContentType: "text/markdown; charset=utf-8",
		Metadata: map[string]string{
			"report_id":   reportID,
			"report_type": input.ReportType,
			"campaign_id": input.CampaignID,
		},
	}); err != nil {
		uc.l.Errorf(ctx, "report.usecase.tryGenerate: Upload failed to bucket %q: %v", uc.config.ReportBucket, err)
		return fmt.Errorf("upload failed to bucket %q: %w", uc.config.ReportBucket, err)
	}

	completedAt := time.Now()
	generationTimeMs := completedAt.Sub(startTime).Milliseconds()

	latest, err := uc.repo.GetReportByID(ctx, reportID)
	if err == nil && latest.Status == report.StatusCancelled {
		uc.l.Infof(ctx, "report.usecase.tryGenerate: Report %s was cancelled, skipping completion update", reportID)
		return nil
	}

	if err := uc.repo.UpdateCompleted(ctx, repository.UpdateCompletedOptions{
		ReportID:          reportID,
		FileURL:           objectName,
		FileSizeBytes:     int64(len(fileBytes)),
		FileFormat:        "md",
		TotalDocsAnalyzed: totalDocs,
		SectionsCount:     sectionsCount,
		GenerationTimeMs:  generationTimeMs,
		CompletedAt:       completedAt,
	}); err != nil {
		uc.l.Errorf(ctx, "report.usecase.tryGenerate: Failed to update completed status: %v", err)
		return fmt.Errorf("update completed: %w", err)
	}

	uc.l.Infof(ctx, "report.usecase.tryGenerate: Report %s completed in %dms", reportID, generationTimeMs)
	return nil
}

// aggregateDocs searches for relevant documents using the search UseCase.
func (uc *implUseCase) aggregateDocs(ctx context.Context, input report.GenerateInput) (search.SearchOutput, error) {
	sc := model.Scope{UserID: input.UserID} // carry caller identity so RBAC pass-through works

	searchInput := search.SearchInput{
		CampaignID: input.CampaignID,
		Query:      buildReportRetrievalQuery(input.ReportType, input.Filters),
		Limit:      uc.config.MaxDocs,
		MinScore:   0.20,
		Filters: search.SearchFilters{
			Sentiments: input.Filters.Sentiments,
			Aspects:    input.Filters.Aspects,
			Platforms:  input.Filters.Platforms,
			DateFrom:   input.Filters.DateFrom,
			DateTo:     input.Filters.DateTo,
			RiskLevels: input.Filters.RiskLevels,
		},
	}

	output, err := uc.searchUC.Search(ctx, sc, searchInput)
	if err != nil {
		return output, err
	}
	return sanitizeReportSearchOutput(output), nil
}

func (uc *implUseCase) ensureReportBucket(ctx context.Context) error {
	bucket := strings.TrimSpace(uc.config.ReportBucket)
	if bucket == "" {
		return fmt.Errorf("report bucket is empty")
	}

	exists, err := uc.minio.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("check report bucket %q: %w", bucket, err)
	}
	if exists {
		return nil
	}

	if err := uc.minio.CreateBucket(ctx, bucket); err != nil {
		existsAfterCreate, checkErr := uc.minio.BucketExists(ctx, bucket)
		if checkErr == nil && existsAfterCreate {
			return nil
		}
		return fmt.Errorf("create report bucket %q: %w", bucket, err)
	}

	uc.l.Infof(ctx, "report.usecase.ensureReportBucket: Created report bucket %s", bucket)
	return nil
}

func (uc *implUseCase) loadReportAnalyticsSummary(ctx context.Context, campaignID string) string {
	if uc.analytics == nil {
		return ""
	}
	snapshotCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	snapshot, err := uc.analytics.Snapshot(snapshotCtx, campaignID)
	if err != nil {
		uc.l.Warnf(ctx, "report.usecase.loadReportAnalyticsSummary: analytics snapshot failed: %v", err)
		return ""
	}
	if !snapshot.HasData() {
		return ""
	}
	return formatAnalyticsSnapshotForReport(snapshot)
}

// buildAggregateQuery generates the search query based on report type.
func buildAggregateQuery(reportType string) string {
	switch reportType {
	case report.ReportTypeSummary:
		return "phản hồi khách hàng mạng xã hội khiếu nại khen chê dịch vụ app tài xế phí giao hàng COD hủy đơn hỗ trợ"
	case report.ReportTypeComparison:
		return "so sánh phản hồi khách hàng theo nền tảng YouTube TikTok Facebook sentiment chủ đề tiêu cực tích cực"
	case report.ReportTypeTrend:
		return "xu hướng thay đổi cảm xúc khách hàng spike momentum chủ đề tăng nhanh khiếu nại khen chê theo thời gian"
	case report.ReportTypeAspectDeep:
		return "phân tích sâu chủ đề khách hàng dịch vụ app tài xế giá phí hỗ trợ hủy đơn COD khiếu nại"
	default:
		return "phân tích phản hồi khách hàng mạng xã hội chủ đề cảm xúc và hành động marketing"
	}
}
