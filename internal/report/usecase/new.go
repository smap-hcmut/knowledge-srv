package usecase

import (
	"knowledge-srv/internal/report"
	"knowledge-srv/internal/report/repository"
	"knowledge-srv/internal/search"

	"github.com/smap-hcmut/shared-libs/go/llm"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/minio"
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
	repo      repository.PostgresRepository
	searchUC  search.UseCase
	llm       llm.LLM
	minio     minio.MinIO
	l         log.Logger
	config    Config
	reportSem chan struct{}
}

// New creates a new report UseCase implementation.
func New(
	repo repository.PostgresRepository,
	searchUC search.UseCase,
	llmClient llm.LLM,
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
		repo:      repo,
		searchUC:  searchUC,
		llm:       llmClient,
		minio:     minioClient,
		l:         l,
		config:    cfg,
		reportSem: make(chan struct{}, 5),
	}
}
