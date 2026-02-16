-- =====================================================
-- Migration: 007 - Create reports table
-- Purpose: Tracking metadata và status của reports
-- Domain: Report (Async Report Generation)
-- Created: 2026-02-16
-- =====================================================

CREATE TABLE IF NOT EXISTS schema_knowledge.reports (
    -- Identity
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id     UUID NOT NULL,              -- Thuộc campaign nào
    user_id         UUID NOT NULL,              -- Ai yêu cầu generate

    -- Report Configuration
    title           VARCHAR(500),               -- Tiêu đề report (auto-generated hoặc user đặt)
    report_type     VARCHAR(50) NOT NULL,       -- 'SUMMARY' | 'COMPARISON' | 'TREND' | 'ASPECT_DEEP_DIVE'
    params_hash     VARCHAR(64) NOT NULL,       -- SHA-256 hash of (campaign_id + report_type + filters)
    filters         JSONB,                      -- Search filters đã dùng: {sentiments, aspects, date_range, ...}

    -- Status Tracking
    status          VARCHAR(20) NOT NULL        -- PROCESSING | COMPLETED | FAILED
                    DEFAULT 'PROCESSING',
    error_message   TEXT,                       -- Lỗi nếu status = FAILED

    -- Output
    file_url        TEXT,                       -- MinIO path: s3://smap-reports/{id}.pdf
    file_size_bytes BIGINT,                     -- Kích thước file (bytes)
    file_format     VARCHAR(10) DEFAULT 'pdf',  -- 'pdf' | 'md'

    -- Performance Metrics
    total_docs_analyzed INT,                    -- Tổng số documents đã phân tích
    sections_count      INT,                    -- Số sections trong report
    generation_time_ms  BIGINT,                 -- Tổng thời gian generate (ms)

    -- Timestamps
    completed_at    TIMESTAMPTZ,                -- Thời điểm hoàn tất
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_reports_campaign ON schema_knowledge.reports(campaign_id);
CREATE INDEX idx_reports_user ON schema_knowledge.reports(user_id);
CREATE INDEX idx_reports_status ON schema_knowledge.reports(status);
CREATE INDEX idx_reports_params_hash ON schema_knowledge.reports(params_hash, status);
CREATE INDEX idx_reports_created ON schema_knowledge.reports(created_at DESC);
