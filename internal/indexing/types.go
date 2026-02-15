package indexing

import "time"

// ============================================
// UseCase Input/Output Types
// ============================================

// ProcessDocumentInput is the input for processing a document
type ProcessDocumentInput struct {
	DocumentID      string
	KnowledgeBaseID string
	FilePath        string
	FileType        string
	ChunkSize       int
	ChunkOverlap    int
}

// ============================================
// Event Types (from Kafka)
// ============================================

// KnowledgeBaseCreatedEvent is published when a knowledge base is created
type KnowledgeBaseCreatedEvent struct {
	KnowledgeBaseID string    `json:"knowledge_base_id"`
	ProjectID       string    `json:"project_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	CreatedBy       string    `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
}

// KnowledgeBaseUpdatedEvent is published when a knowledge base is updated
type KnowledgeBaseUpdatedEvent struct {
	KnowledgeBaseID string    `json:"knowledge_base_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	UpdatedBy       string    `json:"updated_by"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// KnowledgeBaseDeletedEvent is published when a knowledge base is deleted
type KnowledgeBaseDeletedEvent struct {
	KnowledgeBaseID string    `json:"knowledge_base_id"`
	DeletedBy       string    `json:"deleted_by"`
	DeletedAt       time.Time `json:"deleted_at"`
}

// ============================================
// Producer Message Types
// ============================================

// IndexingResult is published when document indexing completes
type IndexingResult struct {
	DocumentID      string    `json:"document_id"`
	KnowledgeBaseID string    `json:"knowledge_base_id"`
	Status          string    `json:"status"` // success, failed
	ChunksIndexed   int       `json:"chunks_indexed"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	CompletedAt     time.Time `json:"completed_at"`
}

// IndexingProgress is published during document indexing
type IndexingProgress struct {
	DocumentID      string    `json:"document_id"`
	KnowledgeBaseID string    `json:"knowledge_base_id"`
	Progress        int       `json:"progress"` // 0-100
	CurrentStep     string    `json:"current_step"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ============================================
// Domain Models
// ============================================

// Document represents a document in the knowledge base
type Document struct {
	ID              string
	KnowledgeBaseID string
	FileName        string
	FilePath        string
	FileType        string
	FileSize        int64
	Status          string // pending, processing, completed, failed
	ChunksCount     int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// KnowledgeBase represents a knowledge base
type KnowledgeBase struct {
	ID          string
	ProjectID   string
	Name        string
	Description string
	Status      string
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Chunk represents a text chunk from a document
type Chunk struct {
	ID              string
	DocumentID      string
	KnowledgeBaseID string
	Content         string
	Embedding       []float32
	Metadata        map[string]interface{}
	ChunkIndex      int
	CreatedAt       time.Time
}
