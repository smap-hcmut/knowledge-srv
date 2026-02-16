-- =====================================================
-- Migration: 006 - Create messages table
-- Purpose: Store individual messages within conversations
-- Domain: Chat (Message Storage)
-- Created: 2026-02-16
-- =====================================================

-- =====================================================
-- Table: messages
-- Purpose: Store user and assistant messages with metadata
-- =====================================================
CREATE TABLE IF NOT EXISTS schema_knowledge.messages (
    -- Identity
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id   UUID NOT NULL               -- FK to conversations.id
                      REFERENCES schema_knowledge.conversations(id)
                      ON DELETE CASCADE,

    -- Message Content
    role              VARCHAR(20) NOT NULL,        -- 'user' | 'assistant'
    content           TEXT NOT NULL,               -- The message text

    -- Assistant Metadata (JSONB, nullable for user messages)
    citations         JSONB,                       -- Array of citation objects
    search_metadata   JSONB,                       -- Search processing metadata
    suggestions       JSONB,                       -- Follow-up suggestion strings
    filters_used      JSONB,                       -- Filters applied during search

    -- Timestamps
    created_at        TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- Indexes
-- =====================================================

-- Primary lookup: List messages by conversation, ordered by time
CREATE INDEX IF NOT EXISTS idx_messages_conversation_created
    ON schema_knowledge.messages(conversation_id, created_at ASC);

-- =====================================================
-- Comments
-- =====================================================
COMMENT ON TABLE schema_knowledge.messages IS
    'Individual messages within chat conversations - stores both user questions and assistant responses with metadata';

COMMENT ON COLUMN schema_knowledge.messages.role IS
    'Message author role: user (human question) or assistant (AI response)';

COMMENT ON COLUMN schema_knowledge.messages.citations IS
    'JSONB array of citation objects extracted from search results (assistant messages only)';

COMMENT ON COLUMN schema_knowledge.messages.search_metadata IS
    'JSONB object with search processing stats: total_docs_searched, docs_used, processing_time_ms, model_used';

COMMENT ON COLUMN schema_knowledge.messages.suggestions IS
    'JSONB array of follow-up question suggestions (assistant messages only)';

COMMENT ON COLUMN schema_knowledge.messages.filters_used IS
    'JSONB object with the search filters that were applied (user messages only)';
