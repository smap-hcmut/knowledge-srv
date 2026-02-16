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
	"knowledge-srv/pkg/paginator"
	"knowledge-srv/pkg/util"
)

// CreateDocument - Insert single record (returns created entity)
func (r *implPostgresRepository) CreateDocument(ctx context.Context, opt repo.CreateDocumentOptions) (model.IndexedDocument, error) {
	dbDoc := &sqlboiler.IndexedDocument{
		AnalyticsID:    opt.AnalyticsID,
		ProjectID:      opt.ProjectID,
		SourceID:       opt.SourceID,
		QdrantPointID:  opt.QdrantPointID,
		CollectionName: opt.CollectionName,
		ContentHash:    opt.ContentHash,
		Status:         opt.Status,
		RetryCount:     null.IntFrom(opt.RetryCount),
		CreatedAt:      null.TimeFrom(time.Now()),
		UpdatedAt:      null.TimeFrom(time.Now()),
	}

	// Handle nullable fields
	if opt.ErrorMessage != nil {
		dbDoc.ErrorMessage = null.StringFrom(*opt.ErrorMessage)
	}
	if opt.BatchID != nil {
		dbDoc.BatchID = null.StringFrom(*opt.BatchID)
	}
	if opt.EmbeddingTimeMs > 0 {
		dbDoc.EmbeddingTimeMS = null.IntFrom(opt.EmbeddingTimeMs)
	}
	if opt.UpsertTimeMs > 0 {
		dbDoc.UpsertTimeMS = null.IntFrom(opt.UpsertTimeMs)
	}
	if opt.TotalTimeMs > 0 {
		dbDoc.TotalTimeMS = null.IntFrom(opt.TotalTimeMs)
	}
	if opt.IndexedAt != nil {
		dbDoc.IndexedAt = null.TimeFrom(*opt.IndexedAt)
	}

	if err := dbDoc.Insert(ctx, r.db, boil.Infer()); err != nil {
		r.l.Errorf(ctx, "indexing.repository.postgre.CreateDocument: Failed to insert document: %v", err)
		return model.IndexedDocument{}, repo.ErrFailedToInsert
	}

	if doc := model.NewIndexedDocumentFromDB(dbDoc); doc != nil {
		return *doc, nil
	}
	return model.IndexedDocument{}, nil
}

// DetailDocument - Get by ID only (primary key lookup)
func (r *implPostgresRepository) DetailDocument(ctx context.Context, id string) (model.IndexedDocument, error) {
	dbDoc, err := sqlboiler.FindIndexedDocument(ctx, r.db, id)
	if err == sql.ErrNoRows {
		r.l.Errorf(ctx, "indexing.repository.postgre.DetailDocument: Document not found: %v", err)
		return model.IndexedDocument{}, nil // Not found
	}
	if err != nil {
		r.l.Errorf(ctx, "indexing.repository.postgre.DetailDocument: Failed to get document: %v", err)
		return model.IndexedDocument{}, repo.ErrFailedToGet
	}

	if doc := model.NewIndexedDocumentFromDB(dbDoc); doc != nil {
		return *doc, nil
	}
	return model.IndexedDocument{}, nil
}

// GetOneDocument - Get single record by filters
func (r *implPostgresRepository) GetOneDocument(ctx context.Context, opt repo.GetOneDocumentOptions) (model.IndexedDocument, error) {
	mods := r.buildGetOneQuery(opt)

	dbDoc, err := sqlboiler.IndexedDocuments(mods...).One(ctx, r.db)
	if err == sql.ErrNoRows {
		r.l.Errorf(ctx, "indexing.repository.postgre.GetOneDocument: Document not found: %v", err)
		return model.IndexedDocument{}, nil // Not found
	}
	if err != nil {
		r.l.Errorf(ctx, "indexing.repository.postgre.GetOneDocument: Failed to get document: %v", err)
		return model.IndexedDocument{}, repo.ErrFailedToGet
	}

	if doc := model.NewIndexedDocumentFromDB(dbDoc); doc != nil {
		return *doc, nil
	}
	return model.IndexedDocument{}, nil
}

// GetDocuments - List with pagination (returns data + paginator)
func (r *implPostgresRepository) GetDocuments(ctx context.Context, opt repo.GetDocumentsOptions) ([]model.IndexedDocument, paginator.Paginator, error) {
	// 1. Count total
	countMods := r.buildGetCountQuery(opt)
	total, err := sqlboiler.IndexedDocuments(countMods...).Count(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "indexing.repository.postgre.GetDocuments: Failed to get documents count: %v", err)
		return nil, paginator.Paginator{}, repo.ErrFailedToList
	}

	// 2. Get data
	mods := r.buildGetQuery(opt)
	dbDocs, err := sqlboiler.IndexedDocuments(mods...).All(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "indexing.repository.postgre.GetDocuments: Failed to get documents: %v", err)
		return nil, paginator.Paginator{}, repo.ErrFailedToList
	}

	// 3. Build paginator
	pag := paginator.Paginator{
		Total:       int64(total),
		Count:       int64(len(dbDocs)),
		PerPage:     int64(opt.Limit),
		CurrentPage: (opt.Offset / opt.Limit) + 1,
	}

	return util.MapSlice(dbDocs, model.NewIndexedDocumentFromDB), pag, nil
}

// ListDocuments - List without pagination
func (r *implPostgresRepository) ListDocuments(ctx context.Context, opt repo.ListDocumentsOptions) ([]model.IndexedDocument, error) {
	mods := r.buildListQuery(opt)

	dbDocs, err := sqlboiler.IndexedDocuments(mods...).All(ctx, r.db)
	if err != nil {
		r.l.Errorf(ctx, "indexing.repository.postgre.ListDocuments: Failed to get documents: %v", err)
		return nil, repo.ErrFailedToList
	}

	return util.MapSlice(dbDocs, model.NewIndexedDocumentFromDB), nil
}

// UpsertDocument - Insert or update (returns entity)
func (r *implPostgresRepository) UpsertDocument(ctx context.Context, opt repo.UpsertDocumentOptions) (model.IndexedDocument, error) {
	dbDoc := &sqlboiler.IndexedDocument{
		AnalyticsID:    opt.AnalyticsID,
		ProjectID:      opt.ProjectID,
		SourceID:       opt.SourceID,
		QdrantPointID:  opt.QdrantPointID,
		CollectionName: opt.CollectionName,
		ContentHash:    opt.ContentHash,
		Status:         opt.Status,
		RetryCount:     null.IntFrom(opt.RetryCount),
		UpdatedAt:      null.TimeFrom(time.Now()),
	}

	// Handle nullable fields
	if opt.ErrorMessage != nil {
		dbDoc.ErrorMessage = null.StringFrom(*opt.ErrorMessage)
	}
	if opt.BatchID != nil {
		dbDoc.BatchID = null.StringFrom(*opt.BatchID)
	}
	if opt.EmbeddingTimeMs > 0 {
		dbDoc.EmbeddingTimeMS = null.IntFrom(opt.EmbeddingTimeMs)
	}
	if opt.UpsertTimeMs > 0 {
		dbDoc.UpsertTimeMS = null.IntFrom(opt.UpsertTimeMs)
	}
	if opt.TotalTimeMs > 0 {
		dbDoc.TotalTimeMS = null.IntFrom(opt.TotalTimeMs)
	}
	if opt.IndexedAt != nil {
		dbDoc.IndexedAt = null.TimeFrom(*opt.IndexedAt)
	}

	err := dbDoc.Upsert(ctx, r.db, true,
		[]string{"analytics_id"}, // Conflict columns
		boil.Infer(), boil.Infer())
	if err != nil {
		r.l.Errorf(ctx, "indexing.repository.postgre.UpsertDocument: Failed to upsert document: %v", err)
		return model.IndexedDocument{}, repo.ErrFailedToUpsert
	}

	// Return upserted entity
	if doc := model.NewIndexedDocumentFromDB(dbDoc); doc != nil {
		return *doc, nil
	}
	return model.IndexedDocument{}, nil
}

// UpdateDocumentStatus - Update status and metrics (returns updated entity)
func (r *implPostgresRepository) UpdateDocumentStatus(ctx context.Context, opt repo.UpdateDocumentStatusOptions) (model.IndexedDocument, error) {
	dbDoc, err := sqlboiler.FindIndexedDocument(ctx, r.db, opt.ID)
	if err != nil {
		r.l.Errorf(ctx, "indexing.repository.postgre.UpdateDocumentStatus: Failed to update document status: %v", err)
		return model.IndexedDocument{}, repo.ErrFailedToUpdateStatus
	}

	// Update fields
	dbDoc.Status = opt.Status
	dbDoc.UpdatedAt = null.TimeFrom(time.Now())

	if opt.Metrics.ErrorMessage != "" {
		dbDoc.ErrorMessage = null.StringFrom(opt.Metrics.ErrorMessage)
	}
	if opt.Metrics.RetryCount > 0 {
		dbDoc.RetryCount = null.IntFrom(opt.Metrics.RetryCount)
	}
	if opt.Metrics.IndexedAt != nil {
		dbDoc.IndexedAt = null.TimeFrom(*opt.Metrics.IndexedAt)
	}
	if opt.Metrics.EmbeddingTimeMs > 0 {
		dbDoc.EmbeddingTimeMS = null.IntFrom(opt.Metrics.EmbeddingTimeMs)
	}
	if opt.Metrics.UpsertTimeMs > 0 {
		dbDoc.UpsertTimeMS = null.IntFrom(opt.Metrics.UpsertTimeMs)
	}
	if opt.Metrics.TotalTimeMs > 0 {
		dbDoc.TotalTimeMS = null.IntFrom(opt.Metrics.TotalTimeMs)
	}

	_, err = dbDoc.Update(ctx, r.db, boil.Infer())
	if err != nil {
		r.l.Errorf(ctx, "indexing.repository.postgre.UpdateDocumentStatus: Failed to update document status: %v", err)
		return model.IndexedDocument{}, repo.ErrFailedToUpdateStatus
	}

	// Return updated entity
	if doc := model.NewIndexedDocumentFromDB(dbDoc); doc != nil {
		return *doc, nil
	}
	return model.IndexedDocument{}, nil
}

// CountDocumentsByProject - Get statistics per project
func (r *implPostgresRepository) CountDocumentsByProject(ctx context.Context, projectID string) (repo.DocumentProjectStats, error) {
	// Use raw SQL for aggregation
	query := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'INDEXED') as total_indexed,
			COUNT(*) FILTER (WHERE status = 'FAILED') as total_failed,
			COUNT(*) FILTER (WHERE status = 'PENDING') as total_pending,
			MAX(indexed_at) as last_indexed_at,
			AVG(total_time_ms) FILTER (WHERE status = 'INDEXED') as avg_index_time_ms
		FROM schema_knowledge.indexed_documents
		WHERE project_id = $1
	`

	var stats repo.DocumentProjectStats
	var lastIndexedAt sql.NullTime
	var avgIndexTimeMs sql.NullFloat64

	err := r.db.QueryRowContext(ctx, query, projectID).Scan(
		&stats.TotalIndexed,
		&stats.TotalFailed,
		&stats.TotalPending,
		&lastIndexedAt,
		&avgIndexTimeMs,
	)
	if err != nil {
		r.l.Errorf(ctx, "indexing.repository.postgre.CountDocumentsByProject: Failed to count documents by project: %v", err)
		return repo.DocumentProjectStats{}, repo.ErrFailedToCount
	}

	stats.ProjectID = projectID
	if lastIndexedAt.Valid {
		stats.LastIndexedAt = &lastIndexedAt.Time
	}
	if avgIndexTimeMs.Valid {
		stats.AvgIndexTimeMs = int(avgIndexTimeMs.Float64)
	}

	return stats, nil
}
