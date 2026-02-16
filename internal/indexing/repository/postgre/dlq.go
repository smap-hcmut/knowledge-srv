package postgre

import (
	"context"
	"database/sql"
	"time"

	"github.com/aarondl/null/v8"
	"github.com/aarondl/sqlboiler/v4/boil"

	repo "knowledge-srv/internal/indexing/repository"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/sqlboiler"
	"knowledge-srv/pkg/util"
)

// CreateDLQ - Insert single DLQ record (returns created entity)
func (r *implPostgresRepository) CreateDLQ(ctx context.Context, opt repo.CreateDLQOptions) (model.IndexingDLQ, error) {
	dbDlq := &sqlboiler.IndexingDLQ{
		AnalyticsID:  opt.AnalyticsID,
		ErrorMessage: opt.ErrorMessage,
		ErrorType:    opt.ErrorType,
		RetryCount:   null.IntFrom(opt.RetryCount),
		Resolved:     null.BoolFrom(false),
		CreatedAt:    null.TimeFrom(time.Now()),
		UpdatedAt:    null.TimeFrom(time.Now()),
	}

	// Handle nullable fields
	if opt.BatchID != nil {
		dbDlq.BatchID = null.StringFrom(*opt.BatchID)
	}

	if err := dbDlq.Insert(ctx, r.db, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "indexing.repository.postgre.CreateDLQ: Failed to insert DLQ: %v", err)
		return model.IndexingDLQ{}, repo.ErrFailedToInsert
	}

	if dlq := model.NewIndexingDLQFromDB(dbDlq); dlq != nil {
		return *dlq, nil
	}
	return model.IndexingDLQ{}, nil
}

// GetOneDLQ - Get single DLQ record by filters
func (r *implPostgresRepository) GetOneDLQ(ctx context.Context, opt repo.GetOneDLQOptions) (model.IndexingDLQ, error) {
	mods := r.buildGetOneDLQQuery(opt)

	dbDlq, err := sqlboiler.IndexingDLQS(mods...).One(ctx, r.db)
	if err == sql.ErrNoRows {
		return model.IndexingDLQ{}, nil // Not found
	}
	if err != nil {
		r.l.Errorf(ctx, "indexing.repository.postgre.GetOneDLQ: Failed to get DLQ: %v", err)
		return model.IndexingDLQ{}, repo.ErrFailedToGet
	}

	if dlq := model.NewIndexingDLQFromDB(dbDlq); dlq != nil {
		return *dlq, nil
	}
	return model.IndexingDLQ{}, nil
}

// ListDLQs - List DLQ records (no pagination, for retry jobs)
func (r *implPostgresRepository) ListDLQs(ctx context.Context, opt repo.ListDLQOptions) ([]model.IndexingDLQ, error) {
	mods := r.buildListDLQQuery(opt)

	dbDlqs, err := sqlboiler.IndexingDLQS(mods...).All(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "indexing.repository.postgre.ListDLQs: Failed to get DLQs: %v", err)
		return nil, repo.ErrFailedToList
	}

	return util.MapSlice(dbDlqs, model.NewIndexingDLQFromDB), nil
}

// MarkResolvedDLQ - Mark DLQ record as resolved
func (r *implPostgresRepository) MarkResolvedDLQ(ctx context.Context, id string) error {
	dbDlq, err := sqlboiler.FindIndexingDLQ(ctx, r.db, id)
	if err != nil {
		r.l.Errorf(ctx, "indexing.repository.postgre.MarkResolvedDLQ: Failed to mark DLQ as resolved: %v", err)
		return repo.ErrFailedToMarkResolved
	}

	dbDlq.Resolved = null.BoolFrom(true)
	dbDlq.UpdatedAt = null.TimeFrom(time.Now())

	_, err = dbDlq.Update(ctx, r.db, boil.Infer())
	if err != nil {
		r.l.Errorf(ctx, "indexing.repository.postgre.MarkResolvedDLQ: Failed to update DLQ: %v", err)
		return repo.ErrFailedToMarkResolved
	}

	return nil
}
