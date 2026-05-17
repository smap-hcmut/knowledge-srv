package repository

import (
	"context"

	"knowledge-srv/internal/model"
)

//go:generate mockery --name ReportRepository
type ReportRepository interface {
	CreateReport(ctx context.Context, opts CreateReportOptions) (*model.Report, error)
	GetReportByID(ctx context.Context, id string) (*model.Report, error)
	FindByParamsHash(ctx context.Context, opts FindByParamsHashOptions) (*model.Report, error)
	UpdateCompleted(ctx context.Context, opts UpdateCompletedOptions) error
	UpdateFailed(ctx context.Context, opts UpdateFailedOptions) error
	UpdateProcessing(ctx context.Context, opts UpdateProcessingOptions) error
	UpdateCancelled(ctx context.Context, opts UpdateCancelledOptions) error
	DeleteReport(ctx context.Context, opts DeleteReportOptions) error
	ListReports(ctx context.Context, opts ListReportsOptions) ([]*model.Report, error)
	CountReports(ctx context.Context, opts ListReportsOptions) (int, error)
}

//go:generate mockery --name PostgresRepository
type PostgresRepository interface {
	ReportRepository
}
