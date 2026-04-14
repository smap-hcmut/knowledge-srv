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
	"golang.org/x/sync/errgroup"
)

// generateInBackground runs the full Map-Reduce report generation pipeline.
// This is called in a goroutine and must handle its own errors.
//
// Pipeline: Aggregate → Sample → Generate (LLM per section) → Compile → Upload
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

	// Phase 3: Generate - Parallel LLM generation per section (up to 3 concurrent)
	templates := getTemplates(input.ReportType)
	sections := make([]generatedSection, len(templates))

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(3) // Max 3 concurrent LLM calls

	for i, tmpl := range templates {
		i, tmpl := i, tmpl
		g.Go(func() error {
			prompt := buildSectionPrompt(tmpl, promptData{
				CampaignID:  input.CampaignID,
				ReportType:  input.ReportType,
				Samples:     formatSamples(samples),
				TotalDocs:   totalDocs,
				Aggregation: formatAggregation(searchOutput.Aggregations),
			})

			llmCtx, cancel := context.WithTimeout(gCtx, 3*time.Minute)
			defer cancel()
			content, err := uc.llm.Generate(llmCtx, prompt)
			if err != nil {
				return fmt.Errorf("section '%s': %w", tmpl.Title, err)
			}

			sections[i] = generatedSection{
				Title:   tmpl.Title,
				Content: content,
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		uc.l.Errorf(ctx, "report.usecase.generateInBackground: LLM generation failed: %v", err)
		_ = uc.repo.UpdateFailed(ctx, repository.UpdateFailedOptions{
			ReportID:     reportID,
			ErrorMessage: fmt.Sprintf("LLM generation failed: %v", err),
		})
		return
	}

	uc.l.Infof(ctx, "report.usecase.generateInBackground: Generated %d sections for report %s", len(sections), reportID)

	// Phase 4: Compile - Assemble markdown and upload
	markdown := compileMarkdown(input, sections, totalDocs)

	objectName := fmt.Sprintf("reports/%s.md", reportID)
	fileBytes := []byte(markdown)

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
		uc.l.Errorf(ctx, "report.usecase.generateInBackground: Upload failed: %v", err)
		_ = uc.repo.UpdateFailed(ctx, repository.UpdateFailedOptions{
			ReportID:     reportID,
			ErrorMessage: fmt.Sprintf("upload failed: %v", err),
		})
		return
	}

	// Mark report as completed
	completedAt := time.Now()
	generationTimeMs := completedAt.Sub(startTime).Milliseconds()

	err = uc.repo.UpdateCompleted(ctx, repository.UpdateCompletedOptions{
		ReportID:          reportID,
		FileURL:           objectName,
		FileSizeBytes:     int64(len(fileBytes)),
		FileFormat:        "md",
		TotalDocsAnalyzed: totalDocs,
		SectionsCount:     len(sections),
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
		Query:      buildAggregateQuery(input.ReportType),
		Limit:      uc.config.MaxDocs,
		MinScore:   0.3, // Lower threshold for broader coverage
		Filters: search.SearchFilters{
			Sentiments: input.Filters.Sentiments,
			Aspects:    input.Filters.Aspects,
			Platforms:  input.Filters.Platforms,
			DateFrom:   input.Filters.DateFrom,
			DateTo:     input.Filters.DateTo,
			RiskLevels: input.Filters.RiskLevels,
		},
	}

	return uc.searchUC.Search(ctx, sc, searchInput)
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
		return "tổng quan phân tích tất cả phản hồi khách hàng"
	case report.ReportTypeComparison:
		return "so sánh phản hồi theo nền tảng và khía cạnh khác nhau"
	case report.ReportTypeTrend:
		return "xu hướng thay đổi cảm xúc và phản hồi theo thời gian"
	case report.ReportTypeAspectDeep:
		return "phân tích chi tiết từng khía cạnh sản phẩm dịch vụ"
	default:
		return "phân tích tổng quan dữ liệu"
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
