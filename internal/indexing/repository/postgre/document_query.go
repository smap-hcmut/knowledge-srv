package postgre

import (
	"github.com/aarondl/sqlboiler/v4/queries/qm"

	repo "knowledge-srv/internal/indexing/repository"
	"knowledge-srv/pkg/util"
)

// buildGetOneQuery - Build query for GetOne
func (r *implPostgresRepository) buildGetOneQuery(opt repo.GetOneDocumentOptions) []qm.QueryMod {
	mods := []qm.QueryMod{}

	// Apply ALL provided filters (AND condition)
	// Business logic to choose which filter belongs in UseCase
	if opt.AnalyticsID != "" {
		mods = append(mods, qm.Where("analytics_id = ?", opt.AnalyticsID))
	}
	if opt.ContentHash != "" {
		mods = append(mods, qm.Where("content_hash = ?", opt.ContentHash))
	}

	return mods
}

// buildGetCountQuery - Build count query for Get (without limit/offset)
func (r *implPostgresRepository) buildGetCountQuery(opt repo.GetDocumentsOptions) []qm.QueryMod {
	mods := []qm.QueryMod{}

	// Filters
	if opt.Status != "" {
		mods = append(mods, qm.Where("status = ?", opt.Status))
	}
	if opt.ProjectID != "" {
		mods = append(mods, qm.Where("project_id = ?", opt.ProjectID))
	}
	if opt.BatchID != "" {
		mods = append(mods, qm.Where("batch_id = ?", opt.BatchID))
	}
	if opt.MaxRetry > 0 {
		mods = append(mods, qm.Where("retry_count < ?", opt.MaxRetry))
	}
	if opt.StaleBefore != nil {
		mods = append(mods, qm.Where("created_at < ?", *opt.StaleBefore))
	}
	if len(opt.ErrorTypes) > 0 {
		mods = append(mods, qm.WhereIn("error_message LIKE ANY(ARRAY[?])", util.ToInterfaceSlice(opt.ErrorTypes)...))
	}

	return mods
}

// buildGetQuery - Build query for Get (with pagination)
func (r *implPostgresRepository) buildGetQuery(opt repo.GetDocumentsOptions) []qm.QueryMod {
	// Start with count filters
	mods := r.buildGetCountQuery(opt)

	// Sorting
	if opt.OrderBy != "" {
		mods = append(mods, qm.OrderBy(opt.OrderBy))
	} else {
		mods = append(mods, qm.OrderBy("created_at DESC")) // Default sorting
	}

	// Pagination (REQUIRED for Get)
	if opt.Limit > 0 {
		mods = append(mods, qm.Limit(opt.Limit))
	}
	if opt.Offset > 0 {
		mods = append(mods, qm.Offset(opt.Offset))
	}

	return mods
}

// buildListQuery - Build query for List (without pagination)
func (r *implPostgresRepository) buildListQuery(opt repo.ListDocumentsOptions) []qm.QueryMod {
	mods := []qm.QueryMod{}

	// Filters
	if opt.Status != "" {
		mods = append(mods, qm.Where("status = ?", opt.Status))
	}
	if opt.ProjectID != "" {
		mods = append(mods, qm.Where("project_id = ?", opt.ProjectID))
	}
	if opt.BatchID != "" {
		mods = append(mods, qm.Where("batch_id = ?", opt.BatchID))
	}
	if opt.MaxRetry > 0 {
		mods = append(mods, qm.Where("retry_count < ?", opt.MaxRetry))
	}
	if opt.StaleBefore != nil {
		mods = append(mods, qm.Where("created_at < ?", *opt.StaleBefore))
	}
	if len(opt.ErrorTypes) > 0 {
		mods = append(mods, qm.WhereIn("error_message LIKE ANY(ARRAY[?])", util.ToInterfaceSlice(opt.ErrorTypes)...))
	}

	// Sorting
	if opt.OrderBy != "" {
		mods = append(mods, qm.OrderBy(opt.OrderBy))
	} else {
		mods = append(mods, qm.OrderBy("created_at ASC")) // Default for background jobs
	}

	// Safety limit (optional)
	if opt.Limit > 0 {
		mods = append(mods, qm.Limit(opt.Limit))
	}

	return mods
}
