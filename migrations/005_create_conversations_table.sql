-- =====================================================
-- Migration: 005 - Create conversations table
-- Purpose: Store chat conversation sessions for RAG Q&A
-- Domain: Chat (Conversation Management)
-- Created: 2026-02-16
-- =====================================================

-- =====================================================
-- Table: conversations
-- Purpose: Track chat conversation sessions between users and the RAG system
-- =====================================================
CREATE TABLE IF NOT EXISTS schema_knowledge.conversations (
    -- Identity
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id     VARCHAR(100) NOT NULL,       -- Campaign this conversation belongs to
    user_id         VARCHAR(100) NOT NULL,        -- User who initiated the conversation

    -- Conversation Metadata
    title           VARCHAR(255) NOT NULL,        -- Auto-generated or user-provided title
    status          VARCHAR(20) NOT NULL          -- ACTIVE | ARCHIVED
                    DEFAULT 'ACTIVE',
    message_count   INT NOT NULL DEFAULT 0,       -- Total messages in this conversation

    -- Timestamps
    last_message_at TIMESTAMPTZ,                  -- Timestamp of the last message (nullable)
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- Indexes
-- =====================================================

-- Primary lookup: List conversations by campaign + user
CREATE INDEX IF NOT EXISTS idx_conversations_campaign_user
    ON schema_knowledge.conversations(campaign_id, user_id);

-- Filter by status
CREATE INDEX IF NOT EXISTS idx_conversations_status
    ON schema_knowledge.conversations(status);

-- Sort by last activity
CREATE INDEX IF NOT EXISTS idx_conversations_last_message
    ON schema_knowledge.conversations(last_message_at DESC NULLS LAST);

-- Sort by creation time
CREATE INDEX IF NOT EXISTS idx_conversations_created
    ON schema_knowledge.conversations(created_at DESC);

-- =====================================================
-- Comments
-- =====================================================
COMMENT ON TABLE schema_knowledge.conversations IS
    'Chat conversation sessions for RAG Q&A system - tracks multi-turn conversations between users and the AI assistant';

COMMENT ON COLUMN schema_knowledge.conversations.campaign_id IS
    'Campaign scope - conversations are scoped to a specific campaign for data isolation';

COMMENT ON COLUMN schema_knowledge.conversations.status IS
    'ACTIVE: Ongoing conversation, ARCHIVED: Conversation is archived and read-only';

COMMENT ON COLUMN schema_knowledge.conversations.message_count IS
    'Denormalized count of messages for quick display without JOIN';

COMMENT ON COLUMN schema_knowledge.conversations.last_message_at IS
    'Timestamp of the most recent message, used for sorting conversations by activity';
