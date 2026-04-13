-- =====================================================
-- Migration: 004 - Remove ingestion_method column (if exists)
-- Purpose: Remove unnecessary tracking of data ingestion method
-- Domain: Indexing (Vector Database Tracking)
-- Created: 2026-02-16
-- Note: Column was removed from 001 DDL; this migration is kept
--       for environments that ran the old 001 before the fix.
-- =====================================================

-- Drop the column (IF EXISTS makes this idempotent)
ALTER TABLE knowledge.indexed_documents 
    DROP COLUMN IF EXISTS ingestion_method;

-- =====================================================
-- Notes:
-- - This column was originally used to track whether data came from Kafka or HTTP API
-- - The distinction is not needed for the current implementation
-- - No data migration needed as this is a metadata-only column
-- =====================================================
