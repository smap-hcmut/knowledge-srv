package http

import (
	"errors"

	"knowledge-srv/internal/indexing"
)

type IndexReq struct {
	BatchID     string `json:"batch_id" binding:"required"`
	ProjectID   string `json:"project_id" binding:"required,uuid"`
	FileURL     string `json:"file_url" binding:"required"`
	RecordCount int    `json:"record_count"`
}

func (r IndexReq) validate() error {
	if len(r.FileURL) < 6 || r.FileURL[:5] != "s3://" {
		return errors.New("file_url must start with s3://")
	}
	return nil
}

func (r IndexReq) toInput() indexing.IndexInput {
	return indexing.IndexInput{
		BatchID:     r.BatchID,
		ProjectID:   r.ProjectID,
		FileURL:     r.FileURL,
		RecordCount: r.RecordCount,
	}
}

type IndexResp struct {
	BatchID       string             `json:"batch_id"`
	TotalRecords  int                `json:"total_records"`
	Indexed       int                `json:"indexed"`
	Failed        int                `json:"failed"`
	Skipped       int                `json:"skipped"`
	DurationMs    int64              `json:"duration_ms"`
	FailedRecords []failedRecordResp `json:"failed_records,omitempty"`
}

type failedRecordResp struct {
	AnalyticsID  string `json:"analytics_id"`
	ErrorType    string `json:"error_type"`
	ErrorMessage string `json:"error_message"`
}

func (h *handler) newIndexResp(output indexing.IndexOutput) IndexResp {
	resp := IndexResp{
		BatchID:      output.BatchID,
		TotalRecords: output.TotalRecords,
		Indexed:      output.Indexed,
		Failed:       output.Failed,
		Skipped:      output.Skipped,
		DurationMs:   output.Duration.Milliseconds(),
	}

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
