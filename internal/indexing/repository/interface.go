package repository

import (
	"context"

	"knowledge-srv/internal/model"
	"knowledge-srv/pkg/paginator"
)

//go:generate mockery --name Repository
type Repository interface {
	DocumentRepository
	DLQRepository
}

// DocumentRepository - Operations for indexed_documents table
type DocumentRepository interface {
	GetDocuments(ctx context.Context, opt GetDocumentsOptions) ([]model.IndexedDocument, paginator.Paginator, error)
	DetailDocument(ctx context.Context, id string) (model.IndexedDocument, error)
	ListDocuments(ctx context.Context, opt ListDocumentsOptions) ([]model.IndexedDocument, error)
	GetOneDocument(ctx context.Context, opt GetOneDocumentOptions) (model.IndexedDocument, error)
	CreateDocument(ctx context.Context, opt CreateDocumentOptions) (model.IndexedDocument, error)
	UpdateDocumentStatus(ctx context.Context, opt UpdateDocumentStatusOptions) (model.IndexedDocument, error)
	UpsertDocument(ctx context.Context, opt UpsertDocumentOptions) (model.IndexedDocument, error)
	CountDocumentsByProject(ctx context.Context, projectID string) (DocumentProjectStats, error)
}

// DLQRepository - Operations for indexing_dlq table
type DLQRepository interface {
	CreateDLQ(ctx context.Context, opt CreateDLQOptions) (model.IndexingDLQ, error)
	GetOneDLQ(ctx context.Context, opt GetOneDLQOptions) (model.IndexingDLQ, error)
	ListDLQs(ctx context.Context, opt ListDLQOptions) ([]model.IndexingDLQ, error)
	MarkResolvedDLQ(ctx context.Context, id string) error
}
