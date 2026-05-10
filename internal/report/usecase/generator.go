package usecase

import (
	"bytes"
	"context"
	"fmt"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/report"
	"knowledge-srv/internal/report/repository"
	"knowledge-srv/internal/search"
	"strings"
	"time"

	"github.com/smap-hcmut/shared-libs/go/minio"
)

// generateInBackground runs the report generation pipeline.
// This is called in a goroutine and must handle its own errors.
//
// Pipeline: Aggregate → rank evidence → generate business brief → compile → upload
func (uc *implUseCase) generateInBackground(ctx context.Context, reportID string, input report.GenerateInput) {
	startTime := time.Now()

	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			uc.l.Errorf(ctx, "report.usecase.generateInBackground: panic recovered: %v", r)
			_ = uc.repo.UpdateFailed(ctx, repository.UpdateFailedOptions{
				ReportID:     reportID,
				ErrorMessage: fmt.Sprintf("internal panic: %v", r),
			})
		}
	}()

	uc.l.Infof(ctx, "report.usecase.generateInBackground: Starting generation for report %s", reportID)

	// Phase 1: Aggregate - Search for relevant documents
	searchOutput, err := uc.aggregateDocs(ctx, input)
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.generateInBackground: Aggregate phase failed: %v", err)
		_ = uc.repo.UpdateFailed(ctx, repository.UpdateFailedOptions{
			ReportID:     reportID,
			ErrorMessage: fmt.Sprintf("aggregate failed: %v", err),
		})
		return
	}

	if len(searchOutput.Results) == 0 {
		_ = uc.repo.UpdateFailed(ctx, repository.UpdateFailedOptions{
			ReportID:     reportID,
			ErrorMessage: "no relevant documents found for report generation",
		})
		return
	}

	totalDocs := len(searchOutput.Results)
	uc.l.Infof(ctx, "report.usecase.generateInBackground: Found %d documents for report %s", totalDocs, reportID)

	// Phase 2: Sample - Select representative documents
	samples := uc.sampleDocs(searchOutput.Results)
	evidence := buildBusinessEvidencePack(samples, uc.config.SampleSize)
	if len(evidence) == 0 {
		evidence = buildBusinessEvidencePack(searchOutput.Results, uc.config.SampleSize)
	}

	// Phase 3: Generate - one coherent business report, grounded by evidence IDs.
	prompt := buildBusinessReportPrompt(input, businessPromptData{
		TotalDocs:      totalDocs,
		Aggregation:    formatAggregation(searchOutput.Aggregations),
		Evidence:       formatBusinessEvidenceForPrompt(evidence),
		Sections:       strings.Join(input.Filters.Sections, ", "),
		CompetitorURLs: strings.Join(input.Filters.CompetitorURLs, ", "),
	})

	llmCtx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	content, err := uc.llm.Generate(llmCtx, prompt)
	cancel()
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.generateInBackground: LLM generation failed: %v", err)
		_ = uc.repo.UpdateFailed(ctx, repository.UpdateFailedOptions{
			ReportID:     reportID,
			ErrorMessage: fmt.Sprintf("LLM generation failed: %v", err),
		})
		return
	}
	content = normalizeBusinessReportMarkdown(content)
	sectionsCount := countBusinessSections(content)

	uc.l.Infof(ctx, "report.usecase.generateInBackground: Generated business report with %d sections for report %s", sectionsCount, reportID)

	// Phase 4: Compile - Assemble markdown and upload
	markdown := compileBusinessMarkdown(input, content, evidence, totalDocs)

	objectName := fmt.Sprintf("reports/%s.md", reportID)
	fileBytes := []byte(markdown)

	if err := uc.ensureReportBucket(ctx); err != nil {
		uc.l.Errorf(ctx, "report.usecase.generateInBackground: Storage setup failed for bucket %q: %v", uc.config.ReportBucket, err)
		_ = uc.repo.UpdateFailed(ctx, repository.UpdateFailedOptions{
			ReportID:     reportID,
			ErrorMessage: fmt.Sprintf("storage setup failed: %v", err),
		})
		return
	}

	_, err = uc.minio.UploadFile(ctx, &minio.UploadRequest{
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
	})
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.generateInBackground: Upload failed to bucket %q: %v", uc.config.ReportBucket, err)
		_ = uc.repo.UpdateFailed(ctx, repository.UpdateFailedOptions{
			ReportID:     reportID,
			ErrorMessage: fmt.Sprintf("upload failed to bucket %q: %v", uc.config.ReportBucket, err),
		})
		return
	}

	// Mark report as completed
	completedAt := time.Now()
	generationTimeMs := completedAt.Sub(startTime).Milliseconds()

	latest, err := uc.repo.GetReportByID(ctx, reportID)
	if err == nil && latest.Status == report.StatusCancelled {
		uc.l.Infof(ctx, "report.usecase.generateInBackground: Report %s was cancelled, skipping completion update", reportID)
		return
	}

	err = uc.repo.UpdateCompleted(ctx, repository.UpdateCompletedOptions{
		ReportID:          reportID,
		FileURL:           objectName,
		FileSizeBytes:     int64(len(fileBytes)),
		FileFormat:        "md",
		TotalDocsAnalyzed: totalDocs,
		SectionsCount:     sectionsCount,
		GenerationTimeMs:  generationTimeMs,
		CompletedAt:       completedAt,
	})
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.generateInBackground: Failed to update completed status: %v", err)
		return
	}

	uc.l.Infof(ctx, "report.usecase.generateInBackground: Report %s completed in %dms", reportID, generationTimeMs)
}

// aggregateDocs searches for relevant documents using the search UseCase.
func (uc *implUseCase) aggregateDocs(ctx context.Context, input report.GenerateInput) (search.SearchOutput, error) {
	sc := model.Scope{} // System-level scope for background tasks

	searchInput := search.SearchInput{
		CampaignID: input.CampaignID,
		Query:      buildReportRetrievalQuery(input.ReportType, input.Filters),
		Limit:      uc.config.MaxDocs,
		MinScore:   0.45,
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

// sampleDocs selects representative documents from the full result set.
func (uc *implUseCase) sampleDocs(results []search.SearchResult) []search.SearchResult {
	if len(results) <= uc.config.SampleSize {
		return results
	}

	// Take top-scoring and evenly distributed samples
	step := len(results) / uc.config.SampleSize
	if step < 1 {
		step = 1
	}

	samples := make([]search.SearchResult, 0, uc.config.SampleSize)
	for i := 0; i < len(results) && len(samples) < uc.config.SampleSize; i += step {
		samples = append(samples, results[i])
	}

	return samples
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

// generatedSection holds LLM-generated content for one report section.
type generatedSection struct {
	Title   string
	Content string
}

// compileMarkdown assembles all sections into a final Markdown document.
func compileMarkdown(input report.GenerateInput, sections []generatedSection, totalDocs int) string {
	var sb strings.Builder

	// Header
	title := input.Title
	if title == "" {
		title = fmt.Sprintf("Báo Cáo %s", input.ReportType)
	}
	sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	sb.WriteString(fmt.Sprintf("**Campaign ID:** %s\n\n", input.CampaignID))
	sb.WriteString(fmt.Sprintf("**Loại báo cáo:** %s\n\n", input.ReportType))
	if len(input.Filters.Sections) > 0 {
		sb.WriteString(fmt.Sprintf("**Sections:** %s\n\n", strings.Join(input.Filters.Sections, ", ")))
	}
	if strings.TrimSpace(input.Filters.Prompt) != "" {
		sb.WriteString(fmt.Sprintf("**Yêu cầu:** %s\n\n", input.Filters.Prompt))
	}
	sb.WriteString(fmt.Sprintf("**Tổng số documents phân tích:** %d\n\n", totalDocs))
	sb.WriteString(fmt.Sprintf("**Thời gian tạo:** %s\n\n", time.Now().Format("02/01/2006 15:04")))
	sb.WriteString("---\n\n")

	// Sections
	for _, section := range sections {
		sb.WriteString(fmt.Sprintf("## %s\n\n", section.Title))
		sb.WriteString(section.Content)
		sb.WriteString("\n\n---\n\n")
	}

	// Footer
	sb.WriteString("*Báo cáo được tạo tự động bởi SMAP Knowledge Service.*\n")

	return sb.String()
}
