package kafka

// ============================================
// Kafka Topics
// ============================================

const (
	// Consumer Topics
	TopicDocumentIndexing    = "knowledge.document.indexing"
	TopicKnowledgeBaseEvents = "knowledge.knowledgebase.events"

	// Producer Topics
	TopicIndexingResults  = "knowledge.indexing.results"
	TopicIndexingProgress = "knowledge.indexing.progress"
)

// ============================================
// Consumer Group IDs
// ============================================

const (
	ConsumerGroupDocumentIndexing    = "knowledge-consumer-document-indexing"
	ConsumerGroupKnowledgeBaseEvents = "knowledge-consumer-kb-events"
)

// ============================================
// Event Types (for routing in knowledge base events topic)
// ============================================

const (
	EventTypeKnowledgeBaseCreated = "knowledge_base.created"
	EventTypeKnowledgeBaseUpdated = "knowledge_base.updated"
	EventTypeKnowledgeBaseDeleted = "knowledge_base.deleted"
)
