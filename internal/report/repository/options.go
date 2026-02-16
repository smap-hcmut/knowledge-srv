package repository

import "time"

type CreateReportOptions struct {
	ID         string
	CampaignID string
	UserID     string
	Title      string
	ReportType string
	ParamsHash string
	Filters    []byte // JSON
}

type FindByParamsHashOptions struct {
	ParamsHash string
	Status     string
}

type UpdateCompletedOptions struct {
	ReportID          string
	FileURL           string
	FileSizeBytes     int64
	FileFormat        string
	TotalDocsAnalyzed int
	SectionsCount     int
	GenerationTimeMs  int64
	CompletedAt       time.Time
}

type UpdateFailedOptions struct {
	ReportID     string
	ErrorMessage string
}

type ListReportsOptions struct {
	CampaignID string
	UserID     string
	Status     string
	Limit      int
	Offset     int
}
