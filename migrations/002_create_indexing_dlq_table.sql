-- =====================================================
-- Migration: 002 - Create indexing_dlq table
-- Purpose: Dead Letter Queue for indexing errors
-- Domain: Indexing (Error Tracking & Recovery)
-- Created: 2026-02-16
-- =====================================================

-- =====================================================
-- Table: indexing_dlq
-- Purpose: Store failed records after all retries so that admin can handle them
-- =====================================================
CREATE TABLE IF NOT EXISTS schema_knowledge.indexing_dlq (
    -- Identity
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analytics_id    UUID NOT NULL,              -- The analytics record that failed
    batch_id        VARCHAR(100),               -- The batch this record belongs to (nullable)

    -- Error Details
    raw_payload     JSONB NOT NULL,             -- Store the original record for debugging or replay
    error_message   TEXT NOT NULL,              -- Error details
    error_type      VARCHAR(50) NOT NULL,       -- Type: PARSE_ERROR | EMBEDDING_ERROR | QDRANT_ERROR | DB_ERROR

    -- Retry Management
    retry_count     INT DEFAULT 0,              -- Number of retry attempts
    max_retries     INT DEFAULT 3,              -- Maximum allowed retries

    -- Resolution Status
    resolved        BOOLEAN DEFAULT false,      -- Has the admin handled this error?

    -- Timestamps
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- Indexes
-- =====================================================

-- Primary lookup: Find DLQ entry by analytics_id
CREATE INDEX IF NOT EXISTS idx_indexing_dlq_analytics
    ON schema_knowledge.indexing_dlq(analytics_id);

-- Batch tracking: List errors by batch
CREATE INDEX IF NOT EXISTS idx_indexing_dlq_batch
    ON schema_knowledge.indexing_dlq(batch_id)
    WHERE batch_id IS NOT NULL;

-- Admin dashboard: List unresolved errors
CREATE INDEX IF NOT EXISTS idx_indexing_dlq_unresolved
    ON schema_knowledge.indexing_dlq(resolved, created_at DESC)
    WHERE resolved = false;

-- Error analysis: Group by error type
CREATE INDEX IF NOT EXISTS idx_indexing_dlq_error_type
    ON schema_knowledge.indexing_dlq(error_type, created_at DESC);

-- Retry eligibility: Find records ready for retry
CREATE INDEX IF NOT EXISTS idx_indexing_dlq_retry_eligible
    ON schema_knowledge.indexing_dlq(retry_count, max_retries, resolved)
    WHERE retry_count < max_retries AND resolved = false;

-- =====================================================
-- Comments
-- =====================================================
COMMENT ON TABLE schema_knowledge.indexing_dlq IS 
    'Dead Letter Queue for indexing errors - stores failed records after all retries for admin review';

COMMENT ON COLUMN schema_knowledge.indexing_dlq.raw_payload IS 
    'JSONB containing the original AnalyticsPost record for replay after the bug is fixed';

COMMENT ON COLUMN schema_knowledge.indexing_dlq.error_type IS 
    'Type of error: PARSE_ERROR (JSON parse), EMBEDDING_ERROR (embedding generation), QDRANT_ERROR (upsert failure), DB_ERROR (database error)';

COMMENT ON COLUMN schema_knowledge.indexing_dlq.resolved IS 
    'true = Admin has reviewed and handled this record (fixed, replayed, or decided to skip)';

COMMENT ON INDEX idx_indexing_dlq_unresolved IS 
    'Partial index for admin dashboard to quickly find unresolved errors';

COMMENT ON INDEX idx_indexing_dlq_retry_eligible IS 
    'Partial index for retry job to find records eligible for retry';
