package http

import (
	"errors"

	"knowledge-srv/internal/report"
	pkgErrors "knowledge-srv/pkg/errors"
)

var (
	errReportNotFound      = pkgErrors.NewHTTPError(404, "Report not found")
	errReportNotCompleted  = pkgErrors.NewHTTPError(400, "Report is not completed yet")
	errCampaignRequired    = pkgErrors.NewHTTPError(400, "Campaign ID is required")
	errInvalidReportType   = pkgErrors.NewHTTPError(400, "Invalid report type")
	errGenerationFailed    = pkgErrors.NewHTTPError(500, "Report generation failed")
	errDuplicateProcessing = pkgErrors.NewHTTPError(409, "Report is already being processed")
	errDownloadURLFailed   = pkgErrors.NewHTTPError(500, "Failed to generate download URL")
)

func (h *handler) mapError(err error) error {
	switch {
	case errors.Is(err, report.ErrReportNotFound):
		return errReportNotFound
	case errors.Is(err, report.ErrReportNotCompleted):
		return errReportNotCompleted
	case errors.Is(err, report.ErrCampaignRequired):
		return errCampaignRequired
	case errors.Is(err, report.ErrInvalidReportType):
		return errInvalidReportType
	case errors.Is(err, report.ErrGenerationFailed):
		return errGenerationFailed
	case errors.Is(err, report.ErrDuplicateProcessing):
		return errDuplicateProcessing
	case errors.Is(err, report.ErrDownloadURLFailed):
		return errDownloadURLFailed
	default:
		panic(err)
	}
}
