package postgre

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"knowledge-srv/internal/notebook"
	"knowledge-srv/internal/notebook/repository"
)

type campaignRepo struct {
	db *sql.DB
}

// NewCampaignRepo creates a campaign repository.
func NewCampaignRepo(db *sql.DB) repository.CampaignRepo {
	return &campaignRepo{db: db}
}

func (r *campaignRepo) GetByCampaignAndPeriod(ctx context.Context, campaignID, periodLabel string) (notebook.NotebookInfo, error) {
	const q = `
		SELECT notebook_id, campaign_id, period_label, created_at
		FROM notebook_campaigns
		WHERE campaign_id = $1 AND period_label = $2 AND status = 'ACTIVE'
	`
	var info notebook.NotebookInfo
	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, q, campaignID, periodLabel).Scan(
		&info.ID, &info.CampaignID, &info.PeriodLabel, &createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return notebook.NotebookInfo{}, repository.ErrNotFound
		}
		return notebook.NotebookInfo{}, fmt.Errorf("notebook_campaigns get: %w", err)
	}
	info.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return info, nil
}

func (r *campaignRepo) Create(ctx context.Context, campaignID, periodLabel, notebookID string) error {
	const q = `
		INSERT INTO notebook_campaigns (campaign_id, notebook_id, period_label, status)
		VALUES ($1, $2, $3, 'ACTIVE')
		ON CONFLICT (campaign_id, period_label) DO UPDATE SET
			notebook_id = EXCLUDED.notebook_id,
			updated_at = NOW(),
			status = 'ACTIVE'
	`
	_, err := r.db.ExecContext(ctx, q, campaignID, notebookID, periodLabel)
	if err != nil {
		return fmt.Errorf("notebook_campaigns insert: %w", err)
	}
	return nil
}
