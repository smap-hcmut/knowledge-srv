package report

import (
	"context"
	"knowledge-srv/internal/model"
)

//go:generate mockery --name UseCase
type UseCase interface {
	Generate(ctx context.Context, sc model.Scope, input GenerateInput) (GenerateOutput, error)
	ListReports(ctx context.Context, sc model.Scope, input ListReportsInput) (ListReportsOutput, error)
	GetReport(ctx context.Context, sc model.Scope, input GetReportInput) (ReportOutput, error)
	GetReportProcess(ctx context.Context, sc model.Scope, input GetReportProcessInput) (ReportProcessOutput, error)
	ListReportPosts(ctx context.Context, sc model.Scope, input ListReportPostsInput) (ListReportPostsOutput, error)
	ListPostComments(ctx context.Context, sc model.Scope, input ListPostCommentsInput) (ListPostCommentsOutput, error)
	DownloadReport(ctx context.Context, sc model.Scope, input DownloadReportInput) (DownloadOutput, error)
	CancelReport(ctx context.Context, sc model.Scope, input CancelReportInput) (CancelOutput, error)
	RetryReport(ctx context.Context, sc model.Scope, input RetryReportInput) (RetryOutput, error)
}
