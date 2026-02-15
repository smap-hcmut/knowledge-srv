-- =====================================================
-- Migration: 004 - Remove ingestion_method column
-- Purpose: Remove unnecessary tracking of data ingestion method
-- Domain: Indexing (Vector Database Tracking)
-- Created: 2026-02-16
-- =====================================================

-- =====================================================
-- Drop ingestion_method from indexed_documents table
-- =====================================================

-- Remove the comment first
COMMENT ON COLUMN schema_knowledge.indexed_documents.ingestion_method IS NULL;

-- Drop the column
ALTER TABLE schema_knowledge.indexed_documents 
    DROP COLUMN IF EXISTS ingestion_method;

-- =====================================================
-- Notes:
-- - This column was originally used to track whether data came from Kafka or HTTP API
-- - The distinction is not needed for the current implementation
-- - No data migration needed as this is a metadata-only column
-- =====================================================
