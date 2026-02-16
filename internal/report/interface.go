package report

import (
	"context"
	"knowledge-srv/internal/model"
)

//go:generate mockery --name UseCase
type UseCase interface {
	Generate(ctx context.Context, sc model.Scope, input GenerateInput) (GenerateOutput, error)
	GetReport(ctx context.Context, sc model.Scope, input GetReportInput) (ReportOutput, error)
	DownloadReport(ctx context.Context, sc model.Scope, input DownloadReportInput) (DownloadOutput, error)
}
