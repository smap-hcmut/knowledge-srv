# SMAP Knowledge Service - Technical Documentation

**Version**: 1.0.0  
**Last Updated**: 17/02/2026  
**Status**: ⚠️ Implementation Complete, Testing In Progress

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [API Reference](#api-reference)
4. [Database Schema](#database-schema)
5. [Integration](#integration)
6. [Known Issues](#known-issues)
7. [Roadmap](#roadmap)

---

## Overview

### Purpose

RAG (Retrieval-Augmented Generation) service biến analytics data thành actionable insights thông qua:

- **Vector Search**: Semantic search với multi-criteria filtering
- **RAG Chat**: Multi-turn Q&A với LLM + citations
- **Report Generation**: Async report creation (Map-Reduce pattern)
- **Analytics Indexing**: Real-time data ingestion từ Analytics Service

### Data Flow

```
Analytics Data (Kafka/HTTP)
  → Embedding (Voyage AI)
  → Vector DB (Qdrant)
  → Search + Filter
  → RAG/Report (Gemini LLM)
  → Output (JSON/PDF)
```

### Tech Stack

| Component | Technology     | Purpose                    |
| --------- | -------------- | -------------------------- |
| Language  | Go 1.25+       | Backend                    |
| Framework | Gin            | HTTP routing               |
| Vector DB | Qdrant 1.10+   | Semantic search (1536-dim) |
| Database  | PostgreSQL 15+ | Metadata & history         |
| Cache     | Redis 7+       | 3-tier caching             |
| Queue     | Kafka          | Async ingestion            |
| Storage   | MinIO          | Report files               |
| LLM       | Gemini 1.5-pro | RAG & reports              |
| Embedding | Voyage AI      | Text → vectors             |

---

## Architecture

### Domain Structure

```
┌────────────────────────────────────────────────────┐
│                  HTTP/Kafka Delivery               │
└────┬──────────┬──────────┬──────────┬──────────────┘
     │          │          │          │
┌────▼────┐ ┌──▼────┐ ┌───▼───┐ ┌───▼──────┐
│indexing │ │search │ │ chat  │ │ report   │
│(write)  │ │(read) │ │(RAG)  │ │(async)   │
└─────────┘ └───▲───┘ └───┬───┘ └────┬─────┘
                │         │          │
                └─────────┴──────────┘
                  (use search.UseCase)
```

**Design Patterns**:

- Clean Architecture (Delivery → UseCase → Repository)
- Domain-Driven Design (4 independent domains)
- Dependency Injection (UseCase interfaces)

### 3-Tier Caching Strategy

| Tier | Key Pattern                | TTL    | Purpose             |
| ---- | -------------------------- | ------ | ------------------- |
| 1    | `embedding:{hash}`         | 7 days | Vector cache        |
| 2    | `campaign_projects:{id}`   | 10 min | Campaign resolution |
| 3    | `search:{campaign}:{hash}` | 5 min  | Search results      |

**Performance**:

- Full cache hit: <10ms
- Partial hit: 200-500ms
- Full miss: 500-1200ms

---

## API Reference

### Authentication

- **Method**: JWT (Cookie `smap_auth_token` hoặc Bearer header)
- **Internal APIs**: Service key authentication (`X-Service-Key`)

### Endpoints Summary

| Domain       | Endpoint                            | Method | Auth    | Purpose                 |
| ------------ | ----------------------------------- | ------ | ------- | ----------------------- |
| **Indexing** | `/internal/index`                   | POST   | Service | Index batch từ MinIO    |
|              | `/internal/index/retry`             | POST   | Service | Retry failed records    |
|              | `/internal/index/statistics/:id`    | GET    | Service | Get stats               |
| **Search**   | `/api/v1/search`                    | POST   | JWT     | Semantic search         |
| **Chat**     | `/api/v1/chat`                      | POST   | JWT     | RAG Q&A                 |
|              | `/api/v1/conversations/:id`         | GET    | JWT     | Get history             |
|              | `/api/v1/campaigns/:id/suggestions` | GET    | JWT     | Smart suggestions       |
| **Report**   | `/api/v1/reports/generate`          | POST   | JWT     | Generate report (async) |
|              | `/api/v1/reports/:id`               | GET    | JWT     | Get status              |
|              | `/api/v1/reports/:id/download`      | GET    | JWT     | Download file           |

### Key API Examples

#### 1. Search

```http
POST /api/v1/search
Content-Type: application/json
Authorization: Bearer {token}

{
  "campaign_id": "camp_vf8",
  "query": "VinFast bị phàn nàn về gì?",
  "filters": {
    "sentiments": ["NEGATIVE"],
    "aspects": ["BATTERY"],
    "platforms": ["tiktok"],
    "date_from": 1707577800,
    "limit": 10
  }
}
```

**Response**: Results + aggregations (by_sentiment, by_aspect, by_platform)

#### 2. Chat (RAG)

```http
POST /api/v1/chat
Content-Type: application/json
Authorization: Bearer {token}

{
  "campaign_id": "camp_vf8",
  "conversation_id": "conv_123",  // optional for multi-turn
  "message": "Cụ thể về pin thì sao?",
  "filters": {
    "aspects": ["BATTERY"]
  }
}
```

**Response**: Answer + citations + suggestions + search_metadata

**RAG Pipeline**:

1. Load history (max 20 messages)
2. Search Qdrant (top 10 docs)
3. Build prompt (System + Context + History + Question)
4. Call Gemini LLM
5. Extract citations
6. Generate suggestions
7. Save to PostgreSQL

#### 3. Report Generation

```http
POST /api/v1/reports/generate
Content-Type: application/json
Authorization: Bearer {token}

{
  "campaign_id": "camp_vf8",
  "report_type": "SUMMARY",  // SUMMARY | COMPARISON | TREND | ASPECT_DEEP_DIVE
  "title": "Báo cáo Q1/2026",
  "filters": {
    "date_from": 1704067200,
    "date_to": 1711929600
  }
}
```

**Response**: `{report_id, status: "PROCESSING"}`

**Map-Reduce Pipeline**:

1. Aggregate: Search all matching docs
2. Sample: Select representative docs
3. Generate: LLM per section
4. Compile: Combine to PDF/Markdown
5. Upload: Save to MinIO

---

## Database Schema

### PostgreSQL (Schema: `schema_knowledge`)

#### 1. indexed_documents

Tracking metadata cho indexed records

```sql
CREATE TABLE schema_knowledge.indexed_documents (
    id UUID PRIMARY KEY,
    analytics_id UUID NOT NULL,
    project_id UUID NOT NULL,
    qdrant_point_id UUID NOT NULL,
    content_hash VARCHAR(64) NOT NULL,      -- SHA-256 dedup
    status VARCHAR(20) DEFAULT 'PENDING',   -- PENDING|INDEXED|FAILED
    retry_count INT DEFAULT 0,
    batch_id VARCHAR(100),
    ingestion_method VARCHAR(20) NOT NULL,  -- 'kafka'|'api'
    embedding_time_ms INT,
    indexed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

#### 2. conversations

Multi-turn conversation metadata

```sql
CREATE TABLE schema_knowledge.conversations (
    id UUID PRIMARY KEY,
    campaign_id VARCHAR(100) NOT NULL,
    user_id VARCHAR(100) NOT NULL,
    title VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'ACTIVE',    -- ACTIVE|ARCHIVED
    message_count INT DEFAULT 0,
    last_message_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

#### 3. messages

Individual messages trong conversations

```sql
CREATE TABLE schema_knowledge.messages (
    id UUID PRIMARY KEY,
    conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,              -- 'user'|'assistant'
    content TEXT NOT NULL,
    citations JSONB,                        -- [{id, content, score}]
    search_metadata JSONB,                  -- {docs_searched, processing_time}
    suggestions JSONB,                      -- ["question 1", ...]
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

#### 4. reports

Report generation tracking

```sql
CREATE TABLE schema_knowledge.reports (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL,
    report_type VARCHAR(50) NOT NULL,       -- SUMMARY|COMPARISON|TREND
    params_hash VARCHAR(64) NOT NULL,       -- SHA-256 dedup
    status VARCHAR(20) DEFAULT 'PROCESSING',-- PROCESSING|COMPLETED|FAILED
    file_url TEXT,                          -- MinIO path
    total_docs_analyzed INT,
    generation_time_ms BIGINT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

### Qdrant Collection

**Collection**: `smap_analytics`

- **Vectors**: 1536 dimensions (Voyage AI)
- **Distance**: Cosine similarity
- **Payload**: Full analytics post (sentiment, aspects, keywords, metadata)

---

## Integration

### External Dependencies

| Service           | Method       | Purpose                | Caching |
| ----------------- | ------------ | ---------------------- | ------- |
| Analytics Service | Kafka + HTTP | Provide analyzed data  | -       |
| Project Service   | HTTP         | Campaign → project_ids | 10 min  |
| Voyage AI         | HTTP         | Text → embeddings      | 7 days  |
| Google Gemini     | HTTP         | LLM generation         | -       |
| MinIO             | S3 API       | File storage           | -       |

### Kafka Integration

**Topic**: `analytics.batch.completed`

**Message Format**:

```json
{
  "batch_id": "batch_001",
  "project_id": "proj_vf8",
  "file_url": "s3://smap-analytics/batches/batch_001.jsonl",
  "record_count": 500
}
```

**Consumer**: Downloads file → parses → indexes records

---

## Known Issues

### Critical Issues

#### 1. ⚠️ THIẾU: Conversation Export & Summary

**Status**: NOT IMPLEMENTED

**Missing Features**:

- Export conversation to PDF/DOCX/Markdown
- LLM-generated conversation summary
- Batch export multiple conversations
- Share conversation với shareable link

**Impact**: Poor UX, no audit trail, difficult collaboration

**Priority**: HIGH

**Suggested APIs**:

```
POST /api/v1/conversations/{id}/export
POST /api/v1/conversations/{id}/summarize
POST /api/v1/campaigns/{id}/conversations/export
POST /api/v1/conversations/{id}/share
```

#### 2. Token Window Management

**Issue**: Sử dụng rough estimation `len(prompt) / 4`

**Impact**:

- Có thể vượt 28K token limit → LLM error
- Hoặc cắt context sớm → mất thông tin

**TODO**: Implement proper token counting (tiktoken)

#### 3. Rate Limiting

**Issue**: Không có throttling mechanism

**Impact**:

- Service abuse risk
- Uncontrolled external API costs
- Potential downtime

**TODO**: Redis-based rate limiter (100 req/min search, 20 req/min chat)

### Medium Priority Issues

#### 4. Cache Invalidation

- **Issue**: Invalidate toàn bộ cache thay vì granular
- **Impact**: Giảm cache hit rate không cần thiết

#### 5. Concurrent Indexing

- **Issue**: MaxConcurrency=10 hardcoded
- **Impact**: Không optimal cho batch sizes khác nhau

#### 6. Report Progress Tracking

- **Issue**: Chỉ có PROCESSING/COMPLETED/FAILED
- **Impact**: User không biết progress

#### 7. Monitoring & Observability

- **Issue**: Thiếu Prometheus metrics, distributed tracing
- **Impact**: Khó debug production issues

#### 8. Security

- **Issue**: Không có prompt injection protection
- **Impact**: Security vulnerability

### Low Priority Issues

9. Multi-language support (chỉ có tiếng Việt)
10. Input validation chưa comprehensive
11. Connection pool settings chưa tune
12. Hallucination detection chưa robust
13. Retry strategy chưa có exponential backoff

---

## Roadmap

### Short-term (1-2 months)

- [ ] **Conversation Export & Summary** ⚠️ HIGH PRIORITY
- [ ] Proper token counting
- [ ] Rate limiting middleware
- [ ] Granular cache invalidation
- [ ] Input validation improvements

### Mid-term (3-6 months)

- [ ] Prometheus metrics + Grafana
- [ ] OpenTelemetry tracing
- [ ] Multi-language support
- [ ] Exponential backoff retry
- [ ] WebSocket/SSE real-time updates

### Long-term (6+ months)

- [ ] Auto-scaling
- [ ] ML-based query optimization
- [ ] Advanced hallucination detection
- [ ] Custom report templates
- [ ] A/B testing framework

---

## Performance Metrics

**Note**: Estimates based on code analysis, chưa có actual benchmarks

| Operation               | Latency       | Notes                           |
| ----------------------- | ------------- | ------------------------------- |
| **Indexing**            | ~340ms/record | Embedding: 200ms, Upsert: 100ms |
| **Search (cache hit)**  | <10ms         | Tier 3 cache                    |
| **Search (cache miss)** | 200-1200ms    | Depends on embedding cache      |
| **Chat (first)**        | 2-4s          | Search + LLM                    |
| **Chat (follow-up)**    | 1.5-3s        | With history                    |
| **Report**              | 2-8 min       | Depends on data size            |

**TODO**: Run load tests để có accurate numbers

---

## Configuration

**File**: `config/knowledge-config.yaml`

```yaml
environment:
  name: production

http_server:
  port: 8080

qdrant:
  host: qdrant.smap.com
  port: 6334

postgres:
  host: postgres.smap.com
  dbname: knowledge
  schema: schema_knowledge

gemini:
  model: gemini-1.5-pro
  api_key: ${GEMINI_API_KEY}

voyage:
  api_key: ${VOYAGE_API_KEY}

redis:
  host: redis.smap.com

minio:
  bucket: smap-reports

kafka:
  brokers: [kafka1.smap.com:9092]
  topic: analytics.batch.completed
```

---

## Resources

- **Swagger UI**: http://localhost:8080/swagger/index.html
- **Repository**: [Link to repo]
- **Monitoring**: Grafana dashboard (TBD)
- **On-call**: Slack #knowledge-alerts

---

**Maintained by**: Knowledge Team  
**Contact**: [Your Email]
