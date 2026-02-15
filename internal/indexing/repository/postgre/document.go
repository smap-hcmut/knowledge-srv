package postgre

import (
	"context"
	"fmt"
	"knowledge-srv/internal/indexing"
)

// CreateDocument creates a new document record
func (r *implRepository) CreateDocument(ctx context.Context, doc indexing.Document) error {
	query := `
		INSERT INTO knowledge.documents (
			id, knowledge_base_id, file_name, file_path, file_type, 
			file_size, status, chunks_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(ctx, query,
		doc.ID,
		doc.KnowledgeBaseID,
		doc.FileName,
		doc.FilePath,
		doc.FileType,
		doc.FileSize,
		doc.Status,
		doc.ChunksCount,
		doc.CreatedAt,
		doc.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create document: %w", err)
	}

	return nil
}

// GetDocument retrieves a document by ID
func (r *implRepository) GetDocument(ctx context.Context, id string) (indexing.Document, error) {
	query := `
		SELECT id, knowledge_base_id, file_name, file_path, file_type, 
		       file_size, status, chunks_count, created_at, updated_at
		FROM knowledge.documents
		WHERE id = $1
	`

	var doc indexing.Document
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&doc.ID,
		&doc.KnowledgeBaseID,
		&doc.FileName,
		&doc.FilePath,
		&doc.FileType,
		&doc.FileSize,
		&doc.Status,
		&doc.ChunksCount,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	)

	if err != nil {
		return indexing.Document{}, fmt.Errorf("failed to get document: %w", err)
	}

	return doc, nil
}

// UpdateDocumentStatus updates the status of a document
func (r *implRepository) UpdateDocumentStatus(ctx context.Context, id string, status string) error {
	query := `
		UPDATE knowledge.documents
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update document status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("document not found: %s", id)
	}

	return nil
}

// ListDocumentsByKnowledgeBase lists all documents for a knowledge base
func (r *implRepository) ListDocumentsByKnowledgeBase(ctx context.Context, kbID string) ([]indexing.Document, error) {
	query := `
		SELECT id, knowledge_base_id, file_name, file_path, file_type, 
		       file_size, status, chunks_count, created_at, updated_at
		FROM knowledge.documents
		WHERE knowledge_base_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, kbID)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	defer rows.Close()

	var documents []indexing.Document
	for rows.Next() {
		var doc indexing.Document
		err := rows.Scan(
			&doc.ID,
			&doc.KnowledgeBaseID,
			&doc.FileName,
			&doc.FilePath,
			&doc.FileType,
			&doc.FileSize,
			&doc.Status,
			&doc.ChunksCount,
			&doc.CreatedAt,
			&doc.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		documents = append(documents, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating documents: %w", err)
	}

	return documents, nil
}
