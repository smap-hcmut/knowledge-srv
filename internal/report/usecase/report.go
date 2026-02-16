package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"knowledge-srv/internal/model"
	"knowledge-srv/internal/report"
	"knowledge-srv/internal/report/repository"
	"knowledge-srv/pkg/minio"
)

// Generate creates a new report or returns existing one if already processing/completed.
// Flow: validate → hash params → check dedup → create record → kick off background generation.
func (uc *implUseCase) Generate(ctx context.Context, sc model.Scope, input report.GenerateInput) (report.GenerateOutput, error) {
	// Validate report type
	if !isValidReportType(input.ReportType) {
		return report.GenerateOutput{}, report.ErrInvalidReportType
	}

	if input.CampaignID == "" {
		return report.GenerateOutput{}, report.ErrCampaignRequired
	}

	// Generate params hash for deduplication
	paramsHash, err := uc.generateParamsHash(input)
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.Generate: Failed to generate params hash: %v", err)
		return report.GenerateOutput{}, report.ErrGenerationFailed
	}

	// Check for existing processing report (deduplication)
	existing, err := uc.repo.FindByParamsHash(ctx, repository.FindByParamsHashOptions{
		ParamsHash: paramsHash,
		Status:     report.StatusProcessing,
	})
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.Generate: Failed to check existing report: %v", err)
		return report.GenerateOutput{}, report.ErrGenerationFailed
	}
	if existing != nil {
		return report.GenerateOutput{
			ReportID: existing.ID,
			Status:   existing.Status,
			Message:  "Report is already being generated",
		}, nil
	}

	// Check for recently completed report (reuse within 1 hour)
	completed, err := uc.repo.FindByParamsHash(ctx, repository.FindByParamsHashOptions{
		ParamsHash: paramsHash,
		Status:     report.StatusCompleted,
	})
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.Generate: Failed to check completed report: %v", err)
		return report.GenerateOutput{}, report.ErrGenerationFailed
	}
	if completed != nil && time.Since(completed.CreatedAt) < 1*time.Hour {
		return report.GenerateOutput{
			ReportID: completed.ID,
			Status:   completed.Status,
			Message:  "Report already completed",
		}, nil
	}

	// Serialize filters for storage
	filterJSON, err := json.Marshal(input.Filters)
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.Generate: Failed to marshal filters: %v", err)
		return report.GenerateOutput{}, report.ErrGenerationFailed
	}

	// Create new report record
	reportID := uuid.New().String()
	rpt, err := uc.repo.CreateReport(ctx, repository.CreateReportOptions{
		ID:         reportID,
		CampaignID: input.CampaignID,
		UserID:     sc.UserID,
		Title:      input.Title,
		ReportType: input.ReportType,
		ParamsHash: paramsHash,
		Filters:    filterJSON,
	})
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.Generate: Failed to create report: %v", err)
		return report.GenerateOutput{}, report.ErrGenerationFailed
	}

	// Launch background generation
	go uc.generateInBackground(rpt.ID, input)

	return report.GenerateOutput{
		ReportID: rpt.ID,
		Status:   report.StatusProcessing,
		Message:  "Report generation started",
	}, nil
}

// GetReport returns the current status and metadata of a report.
func (uc *implUseCase) GetReport(ctx context.Context, sc model.Scope, input report.GetReportInput) (report.ReportOutput, error) {
	rpt, err := uc.repo.GetReportByID(ctx, input.ReportID)
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.GetReport: Failed to get report: %v", err)
		return report.ReportOutput{}, report.ErrReportNotFound
	}

	return uc.buildReportOutput(rpt), nil
}

// DownloadReport generates a presigned download URL for a completed report.
func (uc *implUseCase) DownloadReport(ctx context.Context, sc model.Scope, input report.DownloadReportInput) (report.DownloadOutput, error) {
	rpt, err := uc.repo.GetReportByID(ctx, input.ReportID)
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.DownloadReport: Failed to get report: %v", err)
		return report.DownloadOutput{}, report.ErrReportNotFound
	}

	if rpt.Status != report.StatusCompleted {
		return report.DownloadOutput{}, report.ErrReportNotCompleted
	}

	// Generate presigned download URL
	expiry := 30 * time.Minute
	presigned, err := uc.minio.GetPresignedDownloadURL(ctx, &minio.PresignedURLRequest{
		BucketName: uc.config.ReportBucket,
		ObjectName: rpt.FileURL,
		Expiry:     expiry,
	})
	if err != nil {
		uc.l.Errorf(ctx, "report.usecase.DownloadReport: Failed to generate presigned URL: %v", err)
		return report.DownloadOutput{}, report.ErrDownloadURLFailed
	}

	fileName := fmt.Sprintf("report_%s.%s", rpt.ID, rpt.FileFormat)

	return report.DownloadOutput{
		DownloadURL: presigned.URL,
		ExpiresAt:   presigned.ExpiresAt.Format(time.RFC3339),
		FileName:    fileName,
		FileSize:    rpt.FileSizeBytes,
	}, nil
}

// ----------- Private helpers -----------

// isValidReportType checks if the report type is valid.
func isValidReportType(rt string) bool {
	switch rt {
	case report.ReportTypeSummary, report.ReportTypeComparison,
		report.ReportTypeTrend, report.ReportTypeAspectDeep:
		return true
	}
	return false
}

// generateParamsHash creates a SHA-256 hash for deduplication.
func (uc *implUseCase) generateParamsHash(input report.GenerateInput) (string, error) {
	data := map[string]interface{}{
		"campaign_id": input.CampaignID,
		"report_type": input.ReportType,
		"filters":     input.Filters,
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(b)
	return fmt.Sprintf("%x", hash), nil
}

// buildReportOutput converts a model.Report to report.ReportOutput.
func (uc *implUseCase) buildReportOutput(rpt *model.Report) report.ReportOutput {
	output := report.ReportOutput{
		ID:                rpt.ID,
		CampaignID:        rpt.CampaignID,
		UserID:            rpt.UserID,
		Title:             rpt.Title,
		ReportType:        rpt.ReportType,
		Status:            rpt.Status,
		ErrorMessage:      rpt.ErrorMessage,
		FileFormat:        rpt.FileFormat,
		FileSizeBytes:     rpt.FileSizeBytes,
		TotalDocsAnalyzed: rpt.TotalDocsAnalyzed,
		SectionsCount:     rpt.SectionsCount,
		GenerationTimeMs:  rpt.GenerationTimeMs,
		Filters:           rpt.Filters,
		CreatedAt:         rpt.CreatedAt.Format(time.RFC3339),
	}

	if rpt.CompletedAt != nil {
		t := rpt.CompletedAt.Format(time.RFC3339)
		output.CompletedAt = &t
	}

	return output
}
