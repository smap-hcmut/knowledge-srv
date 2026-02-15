package http

import (
	"errors"

	"knowledge-srv/internal/indexing"
)

// =====================================================
// Request DTOs
// =====================================================

// indexByFileReq - Request body cho IndexByFile
type indexByFileReq struct {
	BatchID     string `json:"batch_id" binding:"required"`
	ProjectID   string `json:"project_id" binding:"required,uuid"`
	FileURL     string `json:"file_url" binding:"required"`
	RecordCount int    `json:"record_count"`
}

// validate - Custom validation
func (r indexByFileReq) validate() error {
	// FileURL phải bắt đầu bằng s3://
	if len(r.FileURL) < 6 || r.FileURL[:5] != "s3://" {
		return errors.New("file_url must start with s3://")
	}
	return nil
}

// toInput - Convert to UseCase input
func (r indexByFileReq) toInput() indexing.IndexInput {
	return indexing.IndexInput{
		BatchID:     r.BatchID,
		ProjectID:   r.ProjectID,
		FileURL:     r.FileURL,
		RecordCount: r.RecordCount,
	}
}

// =====================================================
// Response DTOs
// =====================================================

// indexByFileResp - Response cho IndexByFile
type indexByFileResp struct {
	BatchID       string             `json:"batch_id"`
	TotalRecords  int                `json:"total_records"`
	Indexed       int                `json:"indexed"`
	Failed        int                `json:"failed"`
	Skipped       int                `json:"skipped"`
	DurationMs    int64              `json:"duration_ms"`
	FailedRecords []failedRecordResp `json:"failed_records,omitempty"`
}

// failedRecordResp - Chi tiết record lỗi
type failedRecordResp struct {
	AnalyticsID  string `json:"analytics_id"`
	ErrorType    string `json:"error_type"`
	ErrorMessage string `json:"error_message"`
}

// newIndexByFileResp - Convert UseCase output to response
func (h *handler) newIndexByFileResp(output indexing.IndexOutput) indexByFileResp {
	resp := indexByFileResp{
		BatchID:      output.BatchID,
		TotalRecords: output.TotalRecords,
		Indexed:      output.Indexed,
		Failed:       output.Failed,
		Skipped:      output.Skipped,
		DurationMs:   output.Duration.Milliseconds(),
	}

	// Map failed records
	if len(output.FailedRecords) > 0 {
		resp.FailedRecords = make([]failedRecordResp, len(output.FailedRecords))
		for i, fr := range output.FailedRecords {
			resp.FailedRecords[i] = failedRecordResp{
				AnalyticsID:  fr.AnalyticsID,
				ErrorType:    fr.ErrorType,
				ErrorMessage: fr.ErrorMessage,
			}
		}
	}

	return resp
}
