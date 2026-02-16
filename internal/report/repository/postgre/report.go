package postgre

import (
	"context"
	"database/sql"
	"time"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"

	"knowledge-srv/internal/model"
	"knowledge-srv/internal/report/repository"
	"knowledge-srv/internal/sqlboiler"
)

// CreateReport - Insert a new report record.
func (r *implRepository) CreateReport(ctx context.Context, opts repository.CreateReportOptions) (*model.Report, error) {
	dbReport := buildCreateReport(opts)

	if err := dbReport.Insert(ctx, r.db, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "report.repository.postgre.CreateReport: Failed to insert report: %v", err)
		return nil, repository.ErrReportCreateFailed
	}

	return model.NewReportFromDB(dbReport), nil
}

// GetReportByID - Get report by primary key.
func (r *implRepository) GetReportByID(ctx context.Context, id string) (*model.Report, error) {
	dbReport, err := sqlboiler.FindReport(ctx, r.db, id)
	if err == sql.ErrNoRows {
		return nil, repository.ErrReportNotFound
	}
	if err != nil {
		r.l.Errorf(ctx, "report.repository.postgre.GetReportByID: Failed to get report: %v", err)
		return nil, err
	}

	return model.NewReportFromDB(dbReport), nil
}

// FindByParamsHash - Find a report by params_hash and optional status filter.
func (r *implRepository) FindByParamsHash(ctx context.Context, opts repository.FindByParamsHashOptions) (*model.Report, error) {
	mods := r.buildFindByParamsHashQuery(opts)

	dbReport, err := sqlboiler.Reports(mods...).One(ctx, r.db)
	if err == sql.ErrNoRows {
		return nil, nil // Not found is not an error here
	}
	if err != nil {
		r.l.Errorf(ctx, "report.repository.postgre.FindByParamsHash: Failed to find report: %v", err)
		return nil, err
	}

	return model.NewReportFromDB(dbReport), nil
}

// UpdateCompleted - Mark report as COMPLETED with output metadata.
func (r *implRepository) UpdateCompleted(ctx context.Context, opts repository.UpdateCompletedOptions) error {
	dbReport, err := sqlboiler.FindReport(ctx, r.db, opts.ReportID)
	if err != nil {
		r.l.Errorf(ctx, "report.repository.postgre.UpdateCompleted: Failed to find report: %v", err)
		return repository.ErrReportUpdateFailed
	}

	dbReport.Status = "COMPLETED"
	dbReport.FileURL = null.StringFrom(opts.FileURL)
	dbReport.FileSizeBytes = null.Int64From(opts.FileSizeBytes)
	dbReport.FileFormat = null.StringFrom(opts.FileFormat)
	dbReport.TotalDocsAnalyzed = null.IntFrom(opts.TotalDocsAnalyzed)
	dbReport.SectionsCount = null.IntFrom(opts.SectionsCount)
	dbReport.GenerationTimeMS = null.Int64From(opts.GenerationTimeMs)
	dbReport.CompletedAt = null.TimeFrom(opts.CompletedAt)
	dbReport.UpdatedAt = null.TimeFrom(time.Now())

	_, err = dbReport.Update(ctx, r.db, boil.Infer())
	if err != nil {
		r.l.Errorf(ctx, "report.repository.postgre.UpdateCompleted: Failed to update report: %v", err)
		return repository.ErrReportUpdateFailed
	}

	return nil
}

// UpdateFailed - Mark report as FAILED with error message.
func (r *implRepository) UpdateFailed(ctx context.Context, opts repository.UpdateFailedOptions) error {
	dbReport, err := sqlboiler.FindReport(ctx, r.db, opts.ReportID)
	if err != nil {
		r.l.Errorf(ctx, "report.repository.postgre.UpdateFailed: Failed to find report: %v", err)
		return repository.ErrReportUpdateFailed
	}

	dbReport.Status = "FAILED"
	dbReport.ErrorMessage = null.StringFrom(opts.ErrorMessage)
	dbReport.UpdatedAt = null.TimeFrom(time.Now())

	_, err = dbReport.Update(ctx, r.db, boil.Infer())
	if err != nil {
		r.l.Errorf(ctx, "report.repository.postgre.UpdateFailed: Failed to update report: %v", err)
		return repository.ErrReportUpdateFailed
	}

	return nil
}

// ListReports - List reports with filters and pagination.
func (r *implRepository) ListReports(ctx context.Context, opts repository.ListReportsOptions) ([]*model.Report, error) {
	mods := r.buildListReportsQuery(opts)

	dbReports, err := sqlboiler.Reports(mods...).All(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "report.repository.postgre.ListReports: Failed to list reports: %v", err)
		return nil, err
	}

	result := make([]*model.Report, 0, len(dbReports))
	for _, dbReport := range dbReports {
		if rpt := model.NewReportFromDB(dbReport); rpt != nil {
			result = append(result, rpt)
		}
	}

	return result, nil
}
