package usecase

import (
	"context"
	"fmt"
	"strings"

	"knowledge-srv/internal/indexing"
)

// PurgeProject removes every knowledge artifact tied to a project: the
// per-project Qdrant collection (proj_<project_id>) and every
// indexed_documents row. Idempotent — re-running after a partial failure
// finishes the work without double-counting.
//
// Triggered by the project archive/delete flow in project-srv. Keep the
// Qdrant collection cleanup ordered before the Postgres delete so a crash
// mid-cleanup leaves orphan SQL rows pointing at a missing collection
// (recoverable by re-running) rather than orphan Qdrant data we cannot
// locate from SQL.
func (uc *implUseCase) PurgeProject(ctx context.Context, projectID string) (indexing.PurgeProjectOutput, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return indexing.PurgeProjectOutput{}, fmt.Errorf("PurgeProject: project_id is required")
	}

	collectionName := fmt.Sprintf("proj_%s", projectID)
	if err := uc.pointUC.DropCollection(ctx, collectionName); err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.PurgeProject: drop collection %s failed: %v", collectionName, err)
		return indexing.PurgeProjectOutput{}, err
	}

	affected, err := uc.postgreRepo.DeleteDocumentsByProject(ctx, projectID)
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.PurgeProject: delete documents project=%s failed: %v", projectID, err)
		return indexing.PurgeProjectOutput{}, err
	}

	if err := uc.cacheRepo.InvalidateSearchCache(ctx, projectID); err != nil {
		// Cache invalidation is best-effort — stale entries will TTL out.
		uc.l.Warnf(ctx, "indexing.usecase.PurgeProject: invalidate search cache for %s failed: %v", projectID, err)
	}

	uc.l.Infof(ctx, "indexing.usecase.PurgeProject: project=%s collection=%s documents_deleted=%d", projectID, collectionName, affected)
	return indexing.PurgeProjectOutput{
		ProjectID:        projectID,
		CollectionName:   collectionName,
		DocumentsDeleted: affected,
	}, nil
}
