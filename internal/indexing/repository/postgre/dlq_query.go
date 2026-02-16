package postgre

import (
	"github.com/aarondl/sqlboiler/v4/queries/qm"

	repo "knowledge-srv/internal/indexing/repository"
	"knowledge-srv/pkg/util"
)

// buildGetOneDLQQuery - Build query for GetOneDLQ
func (r *implPostgresRepository) buildGetOneDLQQuery(opt repo.GetOneDLQOptions) []qm.QueryMod {
	mods := []qm.QueryMod{}

	// Apply ALL provided filters (AND condition)
	if opt.ID != "" {
		mods = append(mods, qm.Where("id = ?", opt.ID))
	}
	if opt.AnalyticsID != "" {
		mods = append(mods, qm.Where("analytics_id = ?", opt.AnalyticsID))
	}
	if opt.ContentHash != "" {
		mods = append(mods, qm.Where("content_hash = ?", opt.ContentHash))
	}

	return mods
}

// buildListDLQQuery - Build query for ListDLQ
func (r *implPostgresRepository) buildListDLQQuery(opt repo.ListDLQOptions) []qm.QueryMod {
	mods := []qm.QueryMod{}

	// Filters
	if len(opt.ErrorTypes) > 0 {
		mods = append(mods, qm.WhereIn("error_type IN ?", util.ToInterfaceSlice(opt.ErrorTypes)...))
	}
	if opt.ResolvedOnly {
		mods = append(mods, qm.Where("resolved = ?", true))
	}
	if opt.UnresolvedOnly {
		mods = append(mods, qm.Where("resolved = ?", false))
	}

	// Sorting
	if opt.OrderBy != "" {
		mods = append(mods, qm.OrderBy(opt.OrderBy))
	} else {
		mods = append(mods, qm.OrderBy("created_at ASC")) // Default: oldest first
	}

	// Safety limit
	if opt.Limit > 0 {
		mods = append(mods, qm.Limit(opt.Limit))
	}

	return mods
}
