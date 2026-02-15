package postgre

import (
	"context"
	"fmt"
	"knowledge-srv/internal/indexing"
)

// GetKnowledgeBase retrieves a knowledge base by ID
func (r *implRepository) GetKnowledgeBase(ctx context.Context, id string) (indexing.KnowledgeBase, error) {
	query := `
		SELECT id, project_id, name, description, status, 
		       created_by, created_at, updated_at
		FROM knowledge.knowledge_bases
		WHERE id = $1
	`

	var kb indexing.KnowledgeBase
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&kb.ID,
		&kb.ProjectID,
		&kb.Name,
		&kb.Description,
		&kb.Status,
		&kb.CreatedBy,
		&kb.CreatedAt,
		&kb.UpdatedAt,
	)

	if err != nil {
		return indexing.KnowledgeBase{}, fmt.Errorf("failed to get knowledge base: %w", err)
	}

	return kb, nil
}
