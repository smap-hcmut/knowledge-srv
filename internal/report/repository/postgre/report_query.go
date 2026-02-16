package postgre

import (
	"github.com/aarondl/sqlboiler/v4/queries/qm"

	"knowledge-srv/internal/report/repository"
)

// buildFindByParamsHashQuery - Build query for FindByParamsHash.
func (r *implRepository) buildFindByParamsHashQuery(opts repository.FindByParamsHashOptions) []qm.QueryMod {
	mods := []qm.QueryMod{
		qm.Where("params_hash = ?", opts.ParamsHash),
	}

	if opts.Status != "" {
		mods = append(mods, qm.Where("status = ?", opts.Status))
	}

	// Return most recent first
	mods = append(mods, qm.OrderBy("created_at DESC"))

	return mods
}

// buildListReportsQuery - Build query for ListReports.
func (r *implRepository) buildListReportsQuery(opts repository.ListReportsOptions) []qm.QueryMod {
	mods := []qm.QueryMod{}

	if opts.CampaignID != "" {
		mods = append(mods, qm.Where("campaign_id = ?", opts.CampaignID))
	}
	if opts.UserID != "" {
		mods = append(mods, qm.Where("user_id = ?", opts.UserID))
	}
	if opts.Status != "" {
		mods = append(mods, qm.Where("status = ?", opts.Status))
	}

	// Sorting: most recent first
	mods = append(mods, qm.OrderBy("created_at DESC"))

	// Pagination
	if opts.Limit > 0 {
		mods = append(mods, qm.Limit(opts.Limit))
	}
	if opts.Offset > 0 {
		mods = append(mods, qm.Offset(opts.Offset))
	}

	return mods
}
