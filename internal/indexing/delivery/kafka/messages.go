package kafka

import "time"

// ============================================
// Consumer Message DTOs
// ============================================

// DocumentIndexingMessage is the message format for document indexing requests
type DocumentIndexingMessage struct {
	DocumentID      string `json:"document_id"`
	KnowledgeBaseID string `json:"knowledge_base_id"`
	FilePath        string `json:"file_path"`
	FileType        string `json:"file_type"`
	ChunkSize       int    `json:"chunk_size,omitempty"`
	ChunkOverlap    int    `json:"chunk_overlap,omitempty"`
}

// KnowledgeBaseEventMessage is the message format for knowledge base events
type KnowledgeBaseEventMessage struct {
	EventType       string                 `json:"event_type"`
	KnowledgeBaseID string                 `json:"knowledge_base_id"`
	ProjectID       string                 `json:"project_id,omitempty"`
	Name            string                 `json:"name,omitempty"`
	Description     string                 `json:"description,omitempty"`
	UserID          string                 `json:"user_id,omitempty"`
	Timestamp       time.Time              `json:"timestamp"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ============================================
// Producer Message DTOs
// ============================================

// IndexingResultMessage is published when document indexing completes
type IndexingResultMessage struct {
	DocumentID      string    `json:"document_id"`
	KnowledgeBaseID string    `json:"knowledge_base_id"`
	Status          string    `json:"status"` // success, failed
	ChunksIndexed   int       `json:"chunks_indexed"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	CompletedAt     time.Time `json:"completed_at"`
}

// IndexingProgressMessage is published during document indexing
type IndexingProgressMessage struct {
	DocumentID      string    `json:"document_id"`
	KnowledgeBaseID string    `json:"knowledge_base_id"`
	Progress        int       `json:"progress"` // 0-100
	CurrentStep     string    `json:"current_step"`
	UpdatedAt       time.Time `json:"updated_at"`
}
