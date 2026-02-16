package report

import "errors"

var (
	ErrReportNotFound      = errors.New("report not found")
	ErrReportNotCompleted  = errors.New("report is not completed")
	ErrCampaignRequired    = errors.New("campaign_id is required")
	ErrInvalidReportType   = errors.New("invalid report type")
	ErrGenerationFailed    = errors.New("report generation failed")
	ErrDuplicateProcessing = errors.New("duplicate report is already being processed")
	ErrDownloadURLFailed   = errors.New("failed to generate download URL")
)
