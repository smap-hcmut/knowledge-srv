package model

import (
	"encoding/json"
	"time"

	"knowledge-srv/internal/sqlboiler"

	"github.com/aarondl/null/v8"
)

// Report represents a generated report record.
type Report struct {
	ID         string
	CampaignID string
	UserID     string

	// Report Configuration
	Title      string
	ReportType string // SUMMARY | COMPARISON | TREND | ASPECT_DEEP_DIVE
	ParamsHash string
	Filters    json.RawMessage

	// Status
	Status       string // PROCESSING | COMPLETED | FAILED
	ErrorMessage string

	// Output
	FileURL       string
	FileSizeBytes int64
	FileFormat    string

	// Metrics
	TotalDocsAnalyzed int
	SectionsCount     int
	GenerationTimeMs  int64

	// Timestamps
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewReportFromDB converts a SQLBoiler Report to model Report.
func NewReportFromDB(db *sqlboiler.Report) *Report {
	if db == nil {
		return nil
	}

	rpt := &Report{
		ID:         db.ID,
		CampaignID: db.CampaignID,
		UserID:     db.UserID,
		ReportType: db.ReportType,
		ParamsHash: db.ParamsHash,
		Status:     db.Status,
	}

	// Handle nullable string fields
	if db.Title.Valid {
		rpt.Title = db.Title.String
	}
	if db.ErrorMessage.Valid {
		rpt.ErrorMessage = db.ErrorMessage.String
	}
	if db.FileURL.Valid {
		rpt.FileURL = db.FileURL.String
	}
	if db.FileFormat.Valid {
		rpt.FileFormat = db.FileFormat.String
	}

	// Handle nullable JSON
	if db.Filters.Valid {
		rpt.Filters = json.RawMessage(db.Filters.JSON)
	}

	// Handle nullable numeric fields
	if db.FileSizeBytes.Valid {
		rpt.FileSizeBytes = db.FileSizeBytes.Int64
	}
	if db.TotalDocsAnalyzed.Valid {
		rpt.TotalDocsAnalyzed = db.TotalDocsAnalyzed.Int
	}
	if db.SectionsCount.Valid {
		rpt.SectionsCount = db.SectionsCount.Int
	}
	if db.GenerationTimeMS.Valid {
		rpt.GenerationTimeMs = db.GenerationTimeMS.Int64
	}

	// Handle nullable time fields
	if db.CompletedAt.Valid {
		rpt.CompletedAt = &db.CompletedAt.Time
	}
	if db.CreatedAt.Valid {
		rpt.CreatedAt = db.CreatedAt.Time
	}
	if db.UpdatedAt.Valid {
		rpt.UpdatedAt = db.UpdatedAt.Time
	}

	return rpt
}

// ToDBReport converts model Report to SQLBoiler Report.
func (r *Report) ToDBReport() *sqlboiler.Report {
	db := &sqlboiler.Report{
		ID:         r.ID,
		CampaignID: r.CampaignID,
		UserID:     r.UserID,
		ReportType: r.ReportType,
		ParamsHash: r.ParamsHash,
		Status:     r.Status,
	}

	if r.Title != "" {
		db.Title = null.StringFrom(r.Title)
	}
	if r.ErrorMessage != "" {
		db.ErrorMessage = null.StringFrom(r.ErrorMessage)
	}
	if r.FileURL != "" {
		db.FileURL = null.StringFrom(r.FileURL)
	}
	if r.FileFormat != "" {
		db.FileFormat = null.StringFrom(r.FileFormat)
	}
	if len(r.Filters) > 0 && string(r.Filters) != "null" {
		db.Filters = null.JSONFrom(r.Filters)
	}
	if r.FileSizeBytes > 0 {
		db.FileSizeBytes = null.Int64From(r.FileSizeBytes)
	}
	if r.TotalDocsAnalyzed > 0 {
		db.TotalDocsAnalyzed = null.IntFrom(r.TotalDocsAnalyzed)
	}
	if r.SectionsCount > 0 {
		db.SectionsCount = null.IntFrom(r.SectionsCount)
	}
	if r.GenerationTimeMs > 0 {
		db.GenerationTimeMS = null.Int64From(r.GenerationTimeMs)
	}
	if r.CompletedAt != nil {
		db.CompletedAt = null.TimeFrom(*r.CompletedAt)
	}
	db.CreatedAt = null.TimeFrom(r.CreatedAt)
	db.UpdatedAt = null.TimeFrom(r.UpdatedAt)

	return db
}
