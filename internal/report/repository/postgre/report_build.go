package postgre

import (
	"time"

	"github.com/aarondl/null/v8"

	"knowledge-srv/internal/report/repository"
	"knowledge-srv/internal/sqlboiler"
)

// buildCreateReport - Build sqlboiler Report entity from CreateReportOptions.
func buildCreateReport(opts repository.CreateReportOptions) *sqlboiler.Report {
	now := time.Now()

	dbReport := &sqlboiler.Report{
		ID:         opts.ID,
		CampaignID: opts.CampaignID,
		UserID:     opts.UserID,
		ReportType: opts.ReportType,
		ParamsHash: opts.ParamsHash,
		Status:     "PROCESSING",
		CreatedAt:  null.TimeFrom(now),
		UpdatedAt:  null.TimeFrom(now),
	}

	if opts.Title != "" {
		dbReport.Title = null.StringFrom(opts.Title)
	}

	if len(opts.Filters) > 0 {
		dbReport.Filters = null.JSONFrom(opts.Filters)
	}

	return dbReport
}
