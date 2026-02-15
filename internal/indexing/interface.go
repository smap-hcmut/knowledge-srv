package indexing

import (
	"context"
	"knowledge-srv/internal/model"
)

// UseCase defines the business logic interface for indexing domain
type UseCase interface {
	// Document Indexing
	ProcessDocumentIndexing(ctx context.Context, sc model.Scope, input ProcessDocumentInput) error

	// Knowledge Base Events
	HandleKnowledgeBaseCreated(ctx context.Context, event KnowledgeBaseCreatedEvent) error
	HandleKnowledgeBaseUpdated(ctx context.Context, event KnowledgeBaseUpdatedEvent) error
	HandleKnowledgeBaseDeleted(ctx context.Context, event KnowledgeBaseDeletedEvent) error

	// Producer interface
	Producer
}

// Producer defines the interface for publishing indexing events
type Producer interface {
	PublishIndexingResult(ctx context.Context, result IndexingResult) error
	PublishIndexingProgress(ctx context.Context, progress IndexingProgress) error
}

// Repository defines the data access interface for indexing domain
type Repository interface {
	// Document operations
	CreateDocument(ctx context.Context, doc Document) error
	GetDocument(ctx context.Context, id string) (Document, error)
	UpdateDocumentStatus(ctx context.Context, id string, status string) error

	// Knowledge base operations
	GetKnowledgeBase(ctx context.Context, id string) (KnowledgeBase, error)
	ListDocumentsByKnowledgeBase(ctx context.Context, kbID string) ([]Document, error)
}
