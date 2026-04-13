-- =====================================================
-- Migration: 008 - Create notebook integration tables
-- Purpose: Support Maestro NotebookLM integration for hybrid RAG
-- Domain: Notebook (Maestro Session, Campaign Notebooks, Sources)
-- Created: 2026-03-24
-- =====================================================

-- =====================================================
-- Table: maestro_sessions
-- Purpose: Track browser automation sessions with Maestro service
-- =====================================================
CREATE TABLE IF NOT EXISTS knowledge.maestro_sessions (
    -- Identity
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      VARCHAR(255) NOT NULL UNIQUE,   -- Maestro session ID

    -- Session State
    status          VARCHAR(50) NOT NULL             -- ready | busy | tearingDown
                    DEFAULT 'ready',
    pod_name        VARCHAR(255),                    -- K8s pod that owns this session

    -- Timestamps
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    last_used_at    TIMESTAMPTZ,                     -- Last time session was used for an API call
    last_checked_at TIMESTAMPTZ                      -- Last health check timestamp
);

-- =====================================================
-- Table: notebook_campaigns
-- Purpose: Map campaigns to NotebookLM notebooks (one notebook per campaign+period)
-- =====================================================
CREATE TABLE IF NOT EXISTS knowledge.notebook_campaigns (
    -- Identity
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id     VARCHAR(100) NOT NULL,           -- Campaign this notebook belongs to
    notebook_id     VARCHAR(255) NOT NULL,            -- NotebookLM notebook ID from Maestro

    -- Notebook Metadata
    period_label    VARCHAR(20) NOT NULL,             -- e.g. "2026-Q1"
    status          VARCHAR(50) NOT NULL              -- ACTIVE | ARCHIVED | FAILED
                    DEFAULT 'ACTIVE',
    source_count    INT NOT NULL DEFAULT 0,           -- Number of sources uploaded
    last_synced_at  TIMESTAMPTZ,                      -- Last successful source sync

    -- Timestamps
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),

    -- Constraints
    UNIQUE (campaign_id, period_label)
);

-- =====================================================
-- Table: notebook_sources
-- Purpose: Track individual markdown source uploads to NotebookLM
-- =====================================================
CREATE TABLE IF NOT EXISTS knowledge.notebook_sources (
    -- Identity
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id     VARCHAR(100) NOT NULL,            -- Campaign scope
    notebook_id     VARCHAR(255) NOT NULL,            -- Target notebook

    -- Source Metadata
    week_label      VARCHAR(10) NOT NULL,             -- e.g. "2026-W12"
    part_number     INT NOT NULL DEFAULT 1,           -- Part number within a week
    title           TEXT NOT NULL,                     -- Source title sent to Maestro
    post_count      INT NOT NULL DEFAULT 0,           -- Number of posts in this part
    content_hash    VARCHAR(64),                      -- SHA256 hash for dedup

    -- Maestro Job Tracking
    maestro_job_id  VARCHAR(255),                     -- Maestro async job ID
    status          VARCHAR(50) NOT NULL              -- PENDING | UPLOADING | SYNCED | FAILED
                    DEFAULT 'PENDING',
    retry_count     INT NOT NULL DEFAULT 0,
    error_message   TEXT,                             -- Last error message if FAILED

    -- Timestamps
    synced_at       TIMESTAMPTZ,                      -- When source was successfully synced
    created_at      TIMESTAMPTZ DEFAULT NOW(),

    -- Constraints
    UNIQUE (campaign_id, week_label, part_number)
);

-- =====================================================
-- Indexes
-- =====================================================

-- Maestro sessions: lookup by status
CREATE INDEX IF NOT EXISTS idx_maestro_sessions_status
    ON knowledge.maestro_sessions(status);

-- Notebook campaigns: lookup by campaign
CREATE INDEX IF NOT EXISTS idx_notebook_campaigns_campaign
    ON knowledge.notebook_campaigns(campaign_id);

-- Notebook campaigns: lookup active notebooks
CREATE INDEX IF NOT EXISTS idx_notebook_campaigns_status
    ON knowledge.notebook_campaigns(status);

-- Notebook sources: retry failed sources
CREATE INDEX IF NOT EXISTS idx_notebook_sources_status
    ON knowledge.notebook_sources(status);

-- Notebook sources: lookup by campaign
CREATE INDEX IF NOT EXISTS idx_notebook_sources_campaign
    ON knowledge.notebook_sources(campaign_id);

-- Notebook sources: lookup by maestro job for webhook resolution
CREATE INDEX IF NOT EXISTS idx_notebook_sources_maestro_job
    ON knowledge.notebook_sources(maestro_job_id)
    WHERE maestro_job_id IS NOT NULL;

-- =====================================================
-- Comments
-- =====================================================
COMMENT ON TABLE knowledge.maestro_sessions IS
    'Browser automation sessions with Maestro service for NotebookLM integration';

COMMENT ON TABLE knowledge.notebook_campaigns IS
    'Maps campaigns to NotebookLM notebooks - one notebook per campaign per quarter';

COMMENT ON TABLE knowledge.notebook_sources IS
    'Tracks individual markdown source uploads to NotebookLM notebooks via Maestro';

COMMENT ON COLUMN knowledge.notebook_sources.status IS
    'PENDING: Not yet uploaded, UPLOADING: Maestro job in progress, SYNCED: Successfully uploaded, FAILED: Upload failed';

COMMENT ON COLUMN knowledge.notebook_sources.content_hash IS
    'SHA256 hash of markdown content for deduplication - skip re-upload if hash matches';
