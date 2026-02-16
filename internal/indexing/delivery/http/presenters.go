package http

import (
	"errors"
	"time"

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

// --- RetryFailed ---

type RetryFailedReq struct {
	MaxRetryCount int      `json:"max_retry_count"`
	Limit         int      `json:"limit"`
	ErrorTypes    []string `json:"error_types,omitempty"`
}

func (r RetryFailedReq) toInput() indexing.RetryFailedInput {
	return indexing.RetryFailedInput{
		MaxRetryCount: r.MaxRetryCount,
		Limit:         r.Limit,
		ErrorTypes:    r.ErrorTypes,
	}
}

type RetryFailedResp struct {
	TotalRetried int   `json:"total_retried"`
	Succeeded    int   `json:"succeeded"`
	Failed       int   `json:"failed"`
	DurationMs   int64 `json:"duration_ms"`
}

func (h *handler) newRetryFailedResp(output indexing.RetryFailedOutput) RetryFailedResp {
	return RetryFailedResp{
		TotalRetried: output.TotalRetried,
		Succeeded:    output.Succeeded,
		Failed:       output.Failed,
		DurationMs:   output.Duration.Milliseconds(),
	}
}

// --- Reconcile ---

type ReconcileReq struct {
	StaleDurationMinutes int `json:"stale_duration_minutes"`
	Limit                int `json:"limit"`
}

func (r ReconcileReq) toInput() indexing.ReconcileInput {
	return indexing.ReconcileInput{
		StaleDuration: time.Duration(r.StaleDurationMinutes) * time.Minute,
		Limit:         r.Limit,
	}
}

type ReconcileResp struct {
	TotalChecked int   `json:"total_checked"`
	Fixed        int   `json:"fixed"`
	Requeued     int   `json:"requeued"`
	DurationMs   int64 `json:"duration_ms"`
}

func (h *handler) newReconcileResp(output indexing.ReconcileOutput) ReconcileResp {
	return ReconcileResp{
		TotalChecked: output.TotalChecked,
		Fixed:        output.Fixed,
		Requeued:     output.Requeued,
		DurationMs:   output.Duration.Milliseconds(),
	}
}

// --- GetStatistics ---

type StatisticsResp struct {
	ProjectID      string `json:"project_id"`
	TotalIndexed   int    `json:"total_indexed"`
	TotalFailed    int    `json:"total_failed"`
	TotalPending   int    `json:"total_pending"`
	LastIndexedAt  string `json:"last_indexed_at,omitempty"`
	AvgIndexTimeMs int    `json:"avg_index_time_ms"`
}

func (h *handler) newStatisticsResp(output indexing.StatisticOutput) StatisticsResp {
	resp := StatisticsResp{
		ProjectID:      output.ProjectID,
		TotalIndexed:   output.TotalIndexed,
		TotalFailed:    output.TotalFailed,
		TotalPending:   output.TotalPending,
		AvgIndexTimeMs: output.AvgIndexTimeMs,
	}
	if output.LastIndexedAt != nil {
		resp.LastIndexedAt = output.LastIndexedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return resp
}
