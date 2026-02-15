-- =====================================================
-- Migration: 003 - Create indexing statistics view
-- Purpose: Materialized view for indexing statistics & monitoring
-- Domain: Indexing (Monitoring & Observability)
-- Created: 2026-02-16
-- =====================================================

-- =====================================================
-- View: indexing_stats_by_project
-- Purpose: Indexing statistics per project (for dashboard)
-- =====================================================
CREATE OR REPLACE VIEW schema_knowledge.indexing_stats_by_project AS
SELECT
    project_id,
    
    -- Document counts by status
    COUNT(*) FILTER (WHERE status = 'INDEXED') as total_indexed,
    COUNT(*) FILTER (WHERE status = 'FAILED') as total_failed,
    COUNT(*) FILTER (WHERE status = 'PENDING') as total_pending,
    COUNT(*) FILTER (WHERE status = 'RE_INDEXING') as total_reindexing,
    COUNT(*) as total_all,
    
    -- Success rate (percent of documents indexed)
    ROUND(
        COUNT(*) FILTER (WHERE status = 'INDEXED')::numeric / 
        NULLIF(COUNT(*), 0) * 100,
        2
    ) as success_rate_percent,
    
    -- Performance metrics (for INDEXED records)
    ROUND(AVG(total_time_ms) FILTER (WHERE status = 'INDEXED')) as avg_total_time_ms,
    ROUND(AVG(embedding_time_ms) FILTER (WHERE status = 'INDEXED')) as avg_embedding_time_ms,
    ROUND(AVG(upsert_time_ms) FILTER (WHERE status = 'INDEXED')) as avg_upsert_time_ms,
    
    -- Minimum and maximum processing times (for INDEXED records)
    MIN(total_time_ms) FILTER (WHERE status = 'INDEXED') as min_total_time_ms,
    MAX(total_time_ms) FILTER (WHERE status = 'INDEXED') as max_total_time_ms,
    
    -- Timestamps (creation and indexing)
    MAX(indexed_at) as last_indexed_at,
    MIN(created_at) as first_created_at,
    MAX(created_at) as last_created_at,
    
    -- Ingestion method counts
    COUNT(*) FILTER (WHERE ingestion_method = 'kafka') as total_kafka,
    COUNT(*) FILTER (WHERE ingestion_method = 'api') as total_api
    
FROM schema_knowledge.indexed_documents
GROUP BY project_id;

-- =====================================================
-- View: indexing_stats_by_batch
-- Purpose: Indexing statistics per batch (for batch monitoring)
-- =====================================================
CREATE OR REPLACE VIEW schema_knowledge.indexing_stats_by_batch AS
SELECT
    batch_id,
    project_id,
    ingestion_method,
    
    -- Document counts
    COUNT(*) as total_records,
    COUNT(*) FILTER (WHERE status = 'INDEXED') as indexed_count,
    COUNT(*) FILTER (WHERE status = 'FAILED') as failed_count,
    COUNT(*) FILTER (WHERE status = 'PENDING') as pending_count,
    
    -- Success rate (percent of documents indexed)
    ROUND(
        COUNT(*) FILTER (WHERE status = 'INDEXED')::numeric / 
        NULLIF(COUNT(*), 0) * 100,
        2
    ) as success_rate_percent,
    
    -- Average processing time
    ROUND(AVG(total_time_ms)) as avg_processing_time_ms,
    
    -- Batch time information
    MIN(created_at) as batch_started_at,
    MAX(indexed_at) as batch_completed_at,
    
    -- Batch duration in seconds
    EXTRACT(EPOCH FROM (MAX(indexed_at) - MIN(created_at))) as batch_duration_seconds
    
FROM schema_knowledge.indexed_documents
WHERE batch_id IS NOT NULL
GROUP BY batch_id, project_id, ingestion_method;

-- =====================================================
-- View: indexing_error_summary
-- Purpose: Aggregated errors for troubleshooting
-- =====================================================
CREATE OR REPLACE VIEW schema_knowledge.indexing_error_summary AS
SELECT
    error_type,
    COUNT(*) as error_count,
    COUNT(DISTINCT analytics_id) as unique_records,
    COUNT(DISTINCT batch_id) as affected_batches,
    
    -- Resolution status counts
    COUNT(*) FILTER (WHERE resolved = true) as resolved_count,
    COUNT(*) FILTER (WHERE resolved = false) as unresolved_count,
    
    -- Retry statistics
    ROUND(AVG(retry_count), 2) as avg_retry_count,
    MAX(retry_count) as max_retry_count,
    
    -- Error time span
    MIN(created_at) as first_error_at,
    MAX(created_at) as last_error_at,
    
    -- Errors occurred in the last 24 hours
    COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '24 hours') as errors_last_24h
    
FROM schema_knowledge.indexing_dlq
GROUP BY error_type
ORDER BY error_count DESC;

-- =====================================================
-- View: indexing_health_check
-- Purpose: Quick health check for system monitoring
-- =====================================================
CREATE OR REPLACE VIEW schema_knowledge.indexing_health_check AS
SELECT
    -- Total count by document status
    (SELECT COUNT(*) FROM schema_knowledge.indexed_documents WHERE status = 'INDEXED') as total_indexed,
    (SELECT COUNT(*) FROM schema_knowledge.indexed_documents WHERE status = 'FAILED') as total_failed,
    (SELECT COUNT(*) FROM schema_knowledge.indexed_documents WHERE status = 'PENDING') as total_pending,
    
    -- Number of records created in the last hour
    (SELECT COUNT(*) FROM schema_knowledge.indexed_documents 
     WHERE created_at > NOW() - INTERVAL '1 hour') as records_last_hour,
    
    -- Number of records indexed in the last hour
    (SELECT COUNT(*) FROM schema_knowledge.indexed_documents 
     WHERE indexed_at > NOW() - INTERVAL '1 hour') as indexed_last_hour,
    
    -- Pending records that have been pending for more than 10 minutes
    (SELECT COUNT(*) FROM schema_knowledge.indexed_documents 
     WHERE status = 'PENDING' AND created_at < NOW() - INTERVAL '10 minutes') as stale_pending,
    
    -- Number of unresolved errors in DLQ
    (SELECT COUNT(*) FROM schema_knowledge.indexing_dlq WHERE resolved = false) as unresolved_errors,
    
    -- Average processing time for the most recent 1000 indexed records
    (SELECT ROUND(AVG(total_time_ms)) 
     FROM (SELECT total_time_ms FROM schema_knowledge.indexed_documents 
           WHERE status = 'INDEXED' 
           ORDER BY indexed_at DESC 
           LIMIT 1000) recent) as avg_time_ms_recent,
    
    -- Latest indexed timestamp and the current check time
    (SELECT MAX(indexed_at) FROM schema_knowledge.indexed_documents) as last_indexed_at,
    NOW() as checked_at;

-- =====================================================
-- Comments
-- =====================================================
COMMENT ON VIEW schema_knowledge.indexing_stats_by_project IS 
    'Per-project indexing statistics for dashboard and monitoring';

COMMENT ON VIEW schema_knowledge.indexing_stats_by_batch IS 
    'Per-batch statistics to track batch processing performance';

COMMENT ON VIEW schema_knowledge.indexing_error_summary IS 
    'Error aggregation from DLQ for troubleshooting and identifying patterns';

COMMENT ON VIEW schema_knowledge.indexing_health_check IS 
    'Single-row health check for monitoring systems (Prometheus, Grafana)';
