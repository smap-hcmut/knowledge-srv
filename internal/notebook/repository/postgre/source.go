package postgre

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"knowledge-srv/internal/notebook"
	"knowledge-srv/internal/notebook/repository"
)

type sourceRepo struct {
	db *sql.DB
}

// NewSourceRepo creates a source repository.
func NewSourceRepo(db *sql.DB) repository.SourceRepo {
	return &sourceRepo{db: db}
}

func (r *sourceRepo) GetByContentHash(ctx context.Context, campaignID, contentHash string) (notebook.SourceRecord, error) {
	const q = `
		SELECT id::text, campaign_id, notebook_id, week_label, part_number, title, post_count,
		       COALESCE(content_hash, ''), COALESCE(maestro_job_id, ''), status, retry_count
		FROM notebook_sources
		WHERE campaign_id = $1 AND content_hash = $2
		LIMIT 1
	`
	var rec notebook.SourceRecord
	err := r.db.QueryRowContext(ctx, q, campaignID, contentHash).Scan(
		&rec.ID, &rec.CampaignID, &rec.NotebookID, &rec.WeekLabel, &rec.PartNumber,
		&rec.Title, &rec.PostCount, &rec.ContentHash, &rec.MaestroJobID, &rec.Status, &rec.RetryCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return notebook.SourceRecord{}, repository.ErrNotFound
		}
		return notebook.SourceRecord{}, fmt.Errorf("notebook_sources get by hash: %w", err)
	}
	return rec, nil
}

func (r *sourceRepo) CreateUploading(ctx context.Context, in notebook.SourceUpsertInput) error {
	const q = `
		INSERT INTO notebook_sources (
			campaign_id, notebook_id, week_label, part_number, title, post_count,
			content_hash, maestro_job_id, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (campaign_id, week_label, part_number) DO UPDATE SET
			notebook_id = EXCLUDED.notebook_id,
			title = EXCLUDED.title,
			post_count = EXCLUDED.post_count,
			content_hash = EXCLUDED.content_hash,
			maestro_job_id = EXCLUDED.maestro_job_id,
			status = EXCLUDED.status
	`
	_, err := r.db.ExecContext(ctx, q,
		in.CampaignID, in.NotebookID, in.WeekLabel, in.PartNumber, in.Title, in.PostCount,
		in.ContentHash, in.MaestroJobID, in.Status,
	)
	if err != nil {
		return fmt.Errorf("notebook_sources insert: %w", err)
	}
	return nil
}

func (r *sourceRepo) ListFailedRetryable(ctx context.Context, maxRetries int) ([]notebook.SourceRecord, error) {
	const q = `
		SELECT id::text, campaign_id, notebook_id, week_label, part_number, title, post_count,
		       COALESCE(content_hash, ''), COALESCE(maestro_job_id, ''), status, retry_count
		FROM notebook_sources
		WHERE status = 'FAILED' AND retry_count < $1
		ORDER BY created_at ASC
		LIMIT 50
	`
	rows, err := r.db.QueryContext(ctx, q, maxRetries)
	if err != nil {
		return nil, fmt.Errorf("notebook_sources list failed: %w", err)
	}
	defer rows.Close()
	var out []notebook.SourceRecord
	for rows.Next() {
		var rec notebook.SourceRecord
		if err := rows.Scan(
			&rec.ID, &rec.CampaignID, &rec.NotebookID, &rec.WeekLabel, &rec.PartNumber,
			&rec.Title, &rec.PostCount, &rec.ContentHash, &rec.MaestroJobID, &rec.Status, &rec.RetryCount,
		); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *sourceRepo) HasSyncedForCampaign(ctx context.Context, campaignID string) (bool, error) {
	const q = `
		SELECT EXISTS(
			SELECT 1 FROM notebook_sources
			WHERE campaign_id = $1 AND status = 'SYNCED'
			LIMIT 1
		)
	`
	var ok bool
	err := r.db.QueryRowContext(ctx, q, campaignID).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("notebook_sources has synced: %w", err)
	}
	return ok, nil
}

func (r *sourceRepo) UpdateStatusByMaestroJobID(ctx context.Context, maestroJobID, status string, errMsg *string) error {
	const q = `
		UPDATE notebook_sources
		SET status = $1,
		    error_message = COALESCE($2, error_message),
		    synced_at = CASE WHEN $1 = 'SYNCED' THEN NOW() ELSE synced_at END,
		    retry_count = CASE WHEN $1 = 'FAILED' THEN retry_count + 1 ELSE retry_count END
		WHERE maestro_job_id = $3
	`
	res, err := r.db.ExecContext(ctx, q, status, errMsg, maestroJobID)
	if err != nil {
		return fmt.Errorf("notebook_sources update status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return repository.ErrNotFound
	}
	return nil
}
