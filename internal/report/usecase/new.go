package usecase

import (
	"knowledge-srv/internal/report"
	"knowledge-srv/internal/report/repository"
	"knowledge-srv/internal/search"
	"knowledge-srv/pkg/gemini"
	"knowledge-srv/pkg/log"
	"knowledge-srv/pkg/minio"
)

const (
	defaultReportBucket = "smap-reports"
	defaultMaxDocs      = 500
	defaultSampleSize   = 50
)

// Config holds configuration for report generation.
type Config struct {
	ReportBucket string
	MaxDocs      int
	SampleSize   int
}

type implUseCase struct {
	repo     repository.PostgresRepository
	searchUC search.UseCase
	gemini   gemini.IGemini
	minio    minio.MinIO
	l        log.Logger
	config   Config
}

// New creates a new report UseCase implementation.
func New(
	repo repository.PostgresRepository,
	searchUC search.UseCase,
	gemini gemini.IGemini,
	minioClient minio.MinIO,
	l log.Logger,
	cfg Config,
) report.UseCase {
	if cfg.ReportBucket == "" {
		cfg.ReportBucket = defaultReportBucket
	}
	if cfg.MaxDocs <= 0 {
		cfg.MaxDocs = defaultMaxDocs
	}
	if cfg.SampleSize <= 0 {
		cfg.SampleSize = defaultSampleSize
	}

	return &implUseCase{
		repo:     repo,
		searchUC: searchUC,
		gemini:   gemini,
		minio:    minioClient,
		l:        l,
		config:   cfg,
	}
}
