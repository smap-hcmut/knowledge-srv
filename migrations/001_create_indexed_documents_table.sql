-- =====================================================
-- Migration: 001 - Create indexed_documents table
-- Purpose: Track metadata about documents that have been indexed into Qdrant
-- Domain: Indexing (Vector Database Tracking)
-- Created: 2026-02-16
-- =====================================================

-- =====================================================
-- Table: indexed_documents
-- Purpose: Track metadata about analytics posts that have been indexed
-- =====================================================
CREATE TABLE IF NOT EXISTS schema_knowledge.indexed_documents (
    -- Identity
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analytics_id    UUID NOT NULL,              -- Foreign key to analytics.post_analytics.id
    project_id      UUID NOT NULL,              -- Project owning this document
    source_id       UUID NOT NULL,              -- Data source for the document

    -- Qdrant Reference
    qdrant_point_id UUID NOT NULL,              -- Qdrant point ID
    collection_name VARCHAR(100) NOT NULL,      -- Qdrant collection name (e.g., "smap_analytics")

    -- Content Hash (for deduplication)
    content_hash    VARCHAR(64) NOT NULL,       -- SHA-256 hash of the content (for deduplication)

    -- Indexing Status
    status          VARCHAR(20) NOT NULL        -- PENDING | INDEXED | FAILED | RE_INDEXING
                    DEFAULT 'PENDING',
    error_message   TEXT,                       -- Error message if status is FAILED
    retry_count     INT DEFAULT 0,              -- Retry count

    -- Batch Tracking
    batch_id        VARCHAR(100),               -- Optional: batch identifier
    ingestion_method VARCHAR(20) NOT NULL,      -- Ingestion method: 'kafka' or 'api'

    -- Performance Metrics
    embedding_time_ms   INT,                    -- Time spent generating embeddings (ms)
    upsert_time_ms      INT,                    -- Time spent upserting to Qdrant (ms)
    total_time_ms       INT,                    -- Total processing time (ms)

    -- Timestamps
    indexed_at      TIMESTAMPTZ,                -- Timestamp of successful indexing
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- Indexes
-- =====================================================

-- Unique index for fetching documents by analytics_id
CREATE UNIQUE INDEX IF NOT EXISTS idx_indexed_docs_analytics_id
    ON schema_knowledge.indexed_documents(analytics_id);

-- Index to filter documents by project
CREATE INDEX IF NOT EXISTS idx_indexed_docs_project
    ON schema_knowledge.indexed_documents(project_id);

-- Index to filter documents by batch (only non-null batches)
CREATE INDEX IF NOT EXISTS idx_indexed_docs_batch
    ON schema_knowledge.indexed_documents(batch_id)
    WHERE batch_id IS NOT NULL;

-- Index to filter by document status (e.g., pending/failed)
CREATE INDEX IF NOT EXISTS idx_indexed_docs_status
    ON schema_knowledge.indexed_documents(status);

-- Index to deduplicate by content hash
CREATE INDEX IF NOT EXISTS idx_indexed_docs_content_hash
    ON schema_knowledge.indexed_documents(content_hash);

-- Index for time-based queries by creation time (most recent first)
CREATE INDEX IF NOT EXISTS idx_indexed_docs_created
    ON schema_knowledge.indexed_documents(created_at DESC);

-- Partial index for "stale" records stuck in PENDING status
CREATE INDEX IF NOT EXISTS idx_indexed_docs_stale_pending
    ON schema_knowledge.indexed_documents(status, created_at)
    WHERE status = 'PENDING';

-- =====================================================
-- Comments
-- =====================================================
COMMENT ON TABLE schema_knowledge.indexed_documents IS 
    'Track metadata about analytics posts that have been indexed in the Qdrant vector database';

COMMENT ON COLUMN schema_knowledge.indexed_documents.analytics_id IS 
    'Foreign key to analytics.post_analytics.id - unique identifier for the analytics post';

COMMENT ON COLUMN schema_knowledge.indexed_documents.qdrant_point_id IS 
    'The vector point ID in Qdrant (usually matches analytics_id)';

COMMENT ON COLUMN schema_knowledge.indexed_documents.content_hash IS 
    'SHA-256 hash of the document content for duplicate detection across different sources';

COMMENT ON COLUMN schema_knowledge.indexed_documents.status IS 
    'PENDING: Awaiting processing, INDEXED: Successfully indexed, FAILED: Error during indexing, RE_INDEXING: Currently being re-indexed';

COMMENT ON COLUMN schema_knowledge.indexed_documents.ingestion_method IS 
    'How the data was ingested: kafka (from Kafka topic) or api (from HTTP endpoint)';

COMMENT ON INDEX idx_indexed_docs_stale_pending IS 
    'Partial index for reconcile job to find stale documents that are still PENDING';
