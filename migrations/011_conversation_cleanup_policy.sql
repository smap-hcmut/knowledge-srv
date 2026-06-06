-- =====================================================
-- Migration: 011 - Conversation lifecycle cleanup policy
-- Purpose: Auto-archive idle conversations and hard-delete archived ones so
-- knowledge.conversations does not grow unbounded over the project lifetime.
-- Domain: Chat (Conversation Management)
-- =====================================================

-- Bookkeeping column so we can distinguish "user archived this" from "system
-- archived this because it went stale" and so the hard-delete pass can find
-- the right rows in O(log n).
ALTER TABLE knowledge.conversations
    ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_conversations_status_archived_at
    ON knowledge.conversations(status, archived_at DESC NULLS LAST);

-- =====================================================
-- Function: knowledge.cleanup_stale_conversations
-- Purpose: Soft-archive conversations idle > 90d and hard-delete archived
-- conversations older than 365d. Idempotent; safe to run as often as the
-- operator wants.
-- =====================================================
CREATE OR REPLACE FUNCTION knowledge.cleanup_stale_conversations()
RETURNS TABLE (archived BIGINT, deleted BIGINT)
LANGUAGE plpgsql
AS $$
DECLARE
    archived_count BIGINT := 0;
    deleted_count  BIGINT := 0;
BEGIN
    WITH soft_archived AS (
        UPDATE knowledge.conversations
           SET status = 'ARCHIVED',
               archived_at = NOW(),
               updated_at = NOW()
         WHERE status = 'ACTIVE'
           AND COALESCE(last_message_at, created_at) < NOW() - INTERVAL '90 days'
        RETURNING id
    )
    SELECT count(*) INTO archived_count FROM soft_archived;

    WITH hard_deleted AS (
        DELETE FROM knowledge.conversations
         WHERE status = 'ARCHIVED'
           AND COALESCE(archived_at, last_message_at, created_at) < NOW() - INTERVAL '365 days'
        RETURNING id
    )
    SELECT count(*) INTO deleted_count FROM hard_deleted;

    RETURN QUERY SELECT archived_count, deleted_count;
END;
$$;

COMMENT ON FUNCTION knowledge.cleanup_stale_conversations IS
    '90/365-day TTL pass for chat conversations. Wire it through pg_cron or a knowledge-srv background job; called once per day is enough.';
