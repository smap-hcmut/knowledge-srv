package repository

import "errors"

var (
	ErrReportNotFound     = errors.New("repository: report not found")
	ErrReportCreateFailed = errors.New("repository: failed to create report")
	ErrReportUpdateFailed = errors.New("repository: failed to update report")
)
