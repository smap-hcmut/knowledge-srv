-- Migration: Create notebook_chat_jobs table

CREATE TABLE IF NOT EXISTS knowledge.notebook_chat_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL,
    campaign_id     UUID NOT NULL,
    user_message    TEXT NOT NULL,
    maestro_job_id  VARCHAR(255),
    status          VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    notebook_answer TEXT,
    fallback_used   BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ DEFAULT NOW() + INTERVAL '10 minutes'
);

-- Indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_notebook_chat_jobs_conversation_id ON knowledge.notebook_chat_jobs (conversation_id);
CREATE INDEX IF NOT EXISTS idx_notebook_chat_jobs_campaign_id ON knowledge.notebook_chat_jobs (campaign_id);
CREATE INDEX IF NOT EXISTS idx_notebook_chat_jobs_status ON knowledge.notebook_chat_jobs (status);
CREATE INDEX IF NOT EXISTS idx_notebook_chat_jobs_expires_at ON knowledge.notebook_chat_jobs (expires_at);
