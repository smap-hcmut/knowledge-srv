# Knowledge Service Architecture Diagram

## KIẾN TRÚC TỔNG QUAN

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         KNOWLEDGE SERVICE ARCHITECTURE                          │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │                            CLIENT LAYER                                   │  │
│  ├───────────────────────────────────────────────────────────────────────────┤  │
│  │                                                                           │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                   │  │
│  │  │   Web UI     │  │  Mobile App  │  │ Analytics    │                   │  │
│  │  │  (React)     │  │              │  │  Service     │                   │  │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘                   │  │
│  │         │                 │                 │                           │  │
│  │         └─────────────────┴─────────────────┘                           │  │
│  │                           │                                              │  │
│  │                           ↓ HTTP/REST                                    │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                              │                                                  │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │                      KNOWLEDGE SERVICE (Go)                               │  │
│  ├───────────────────────────────────────────────────────────────────────────┤  │
│  │                                                                           │  │
│  │  ┌─────────────────────────────────────────────────────────────────────┐  │  │
│  │  │  API LAYER                                                          │  │  │
│  │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐           │  │  │
│  │  │  │  Chat    │  │  Search  │  │  Report  │  │  Index   │           │  │  │
│  │  │  │ Handler  │  │ Handler  │  │ Handler  │  │ Handler  │           │  │  │
│  │  │  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘           │  │  │
│  │  └───────┼──────────────┼──────────────┼──────────────┼───────────────┘  │  │
│  │          │              │              │              │                  │  │
│  │  ┌───────┼──────────────┼──────────────┼──────────────┼───────────────┐  │  │
│  │  │  BUSINESS LOGIC LAYER                                              │  │  │
│  │  │       │              │              │              │                │  │  │
│  │  │  ┌────▼────┐    ┌────▼────┐   ┌────▼────┐   ┌────▼────┐          │  │  │
│  │  │  │   RAG   │    │ Search  │   │ Report  │   │ Vector  │          │  │  │
│  │  │  │Pipeline │    │ Service │   │Generator│   │Indexer  │          │  │  │
│  │  │  └────┬────┘    └────┬────┘   └────┬────┘   └────┬────┘          │  │  │
│  │  └───────┼──────────────┼──────────────┼──────────────┼───────────────┘  │  │
│  │          │              │              │              │                  │  │
│  └──────────┼──────────────┼──────────────┼──────────────┼──────────────────┘  │
│             │              │              │              │                     │
│  ┌──────────┼──────────────┼──────────────┼──────────────┼──────────────────┐  │
│  │  EXTERNAL DEPENDENCIES (6 BẮT BUỘC)                                      │  │
│  ├──────────┼──────────────┼──────────────┼──────────────┼──────────────────┤  │
│  │          │              │              │              │                  │  │
│  │  ┌───────▼────────┐  ┌──▼──────────┐  ┌▼────────────┐  ┌▼─────────────┐  │  │
│  │  │  1. QDRANT     │  │ 2. OPENAI   │  │3.POSTGRESQL │  │  4. REDIS    │  │  │
│  │  │  Vector DB     │  │  API        │  │  Database   │  │  Cache       │  │  │
│  │  │                │  │             │  │             │  │              │  │  │
│  │  │ • Vectors      │  │ • Embedding │  │ • Metadata  │  │ • Embedding  │  │  │
│  │  │ • Search       │  │ • LLM Gen   │  │ • History   │  │   cache      │  │  │
│  │  │ • Filters      │  │             │  │ • Reports   │  │ • Search     │  │  │
│  │  │                │  │             │  │   metadata  │  │   cache      │  │  │
│  │  └────────────────┘  └─────────────┘  └─────────────┘  │ • Rate limit │  │  │
│  │                                                         └──────────────┘  │  │
│  │                                                                           │  │
│  │  ┌──────────────────┐  ┌──────────────────┐                              │  │
│  │  │ 5. PROJECT       │  │  6. MINIO        │                              │  │
│  │  │    SERVICE       │  │  Object Storage  │                              │  │
│  │  │                  │  │                  │                              │  │
│  │  │ • Get campaign   │  │ • Reports (PDF)  │                              │  │
│  │  │   projects       │  │ • Artifacts      │                              │  │
│  │  │ • Validate       │  │ • Presigned URLs │                              │  │
│  │  │   access         │  │                  │                              │  │
│  │  └──────────────────┘  └──────────────────┘                              │  │
│  │                                                                           │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                                                                 │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │  OPTIONAL DEPENDENCIES (3)                                                │  │
│  ├───────────────────────────────────────────────────────────────────────────┤  │
│  │                                                                           │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                   │  │
│  │  │  7. KAFKA    │  │8. PROMETHEUS │  │  9. JAEGER   │                   │  │
│  │  │  Events      │  │  Metrics     │  │  Tracing     │                   │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘                   │  │
│  │                                                                           │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---


## DATA FLOW CHI TIẾT

### Flow 1: Chat Request

```
┌─────────────────────────────────────────────────────────────────┐
│  USER: "VinFast bị đánh giá tiêu cực về gì?"                    │
│  Campaign: "So sánh Xe điện Q1/2026"                            │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ↓ POST /api/v1/chat
┌─────────────────────────────────────────────────────────────────┐
│  KNOWLEDGE SERVICE                                              │
│                                                                 │
│  Step 1: Get Campaign Projects                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  → Project Service                                      │    │
│  │    GET /api/v1/campaigns/camp_001                       │    │
│  │  ← Response: {project_ids: ["proj_vf8", "proj_byd"]}   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                         ↓                                       │
│  Step 2: Generate Query Embedding                              │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Check Redis Cache:                                     │    │
│  │    Key: embedding:{query_hash}                          │    │
│  │  ├─ HIT → Use cached vector                             │    │
│  │  └─ MISS → Call OpenAI API                              │    │
│  │             POST /v1/embeddings                         │    │
│  │             Model: text-embedding-3-small               │    │
│  │             → vector: [0.123, -0.456, ...]              │    │
│  │             → Cache in Redis (TTL: 7 days)              │    │
│  └─────────────────────────────────────────────────────────┘    │
│                         ↓                                       │
│  Step 3: Search Qdrant                                          │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Check Redis Cache:                                     │    │
│  │    Key: search:{campaign_id}:{query_hash}               │    │
│  │  ├─ HIT → Use cached results                            │    │
│  │  └─ MISS → Query Qdrant                                 │    │
│  │             POST /collections/smap_analytics/points/    │    │
│  │                  search                                 │    │
│  │             Filters:                                    │    │
│  │             - project_id IN ["proj_vf8", "proj_byd"]    │    │
│  │             - sentiment = "NEGATIVE"                    │    │
│  │             Limit: 10                                   │    │
│  │             → Top 10 relevant documents                 │    │
│  │             → Cache in Redis (TTL: 5 min)               │    │
│  └─────────────────────────────────────────────────────────┘    │
│                         ↓                                       │
│  Step 4: Build Context                                          │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Extract content from search results                    │    │
│  │  Format: [1] content1\n[2] content2\n...               │    │
│  └─────────────────────────────────────────────────────────┘    │
│                         ↓                                       │
│  Step 5: Generate Answer                                        │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  → OpenAI API                                           │    │
│  │    POST /v1/chat/completions                            │    │
│  │    Model: gpt-4                                         │    │
│  │    Messages:                                            │    │
│  │    - System: "Bạn là trợ lý phân tích..."              │    │
│  │    - User: Context + Question                          │    │
│  │  ← Response: Generated answer                           │    │
│  └─────────────────────────────────────────────────────────┘    │
│                         ↓                                       │
│  Step 6: Save Conversation                                      │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  → PostgreSQL                                           │    │
│  │    INSERT INTO knowledge.messages                       │    │
│  │    (conversation_id, role, content, citations)          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                         ↓                                       │
│  Step 7: Publish Event (Optional)                              │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  → Kafka                                                │    │
│  │    Topic: knowledge.queries                             │    │
│  │    Event: CHAT_QUERY                                    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ↓ Response
┌─────────────────────────────────────────────────────────────────┐
│  {                                                              │
│    "answer": "VinFast nhận đánh giá tiêu cực về PIN (45%)...", │
│    "citations": [...],                                          │
│    "suggestions": [...]                                         │
│  }                                                              │
└─────────────────────────────────────────────────────────────────┘
```

**Latency Breakdown:**
- Step 1 (Project Service): 50ms
- Step 2 (Embedding): 800ms (or 5ms if cached)
- Step 3 (Qdrant Search): 200ms (or 5ms if cached)
- Step 4 (Context Building): 10ms
- Step 5 (LLM Generation): 4000ms
- Step 6 (Save to DB): 20ms
- Step 7 (Kafka): 5ms (async)

**Total:** ~5s (or ~4.3s with cache hits)

---

### Flow 2: Document Indexing

```
┌─────────────────────────────────────────────────────────────────┐
│  ANALYTICS SERVICE                                              │
│  (Sau khi phân tích xong)                                       │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ↓ POST /internal/index
┌─────────────────────────────────────────────────────────────────┐
│  KNOWLEDGE SERVICE                                              │
│                                                                 │
│  Step 1: Validate Request                                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Check required fields:                                 │    │
│  │  - analytics_id, project_id, content                    │    │
│  │  - sentiment, aspects                                   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                         ↓                                       │
│  Step 2: Generate Embedding                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Check Redis Cache:                                     │    │
│  │    Key: embedding:{content_hash}                        │    │
│  │  ├─ HIT → Use cached vector                             │    │
│  │  └─ MISS → Call OpenAI API                              │    │
│  │             POST /v1/embeddings                         │    │
│  │             → vector: [0.123, -0.456, ...]              │    │
│  │             → Cache in Redis                            │    │
│  └─────────────────────────────────────────────────────────┘    │
│                         ↓                                       │
│  Step 3: Prepare Qdrant Point                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  {                                                      │    │
│  │    "id": "analytics_001",                               │    │
│  │    "vector": [0.123, -0.456, ...],                      │    │
│  │    "payload": {                                         │    │
│  │      "project_id": "proj_vf8",                          │    │
│  │      "content": "...",                                  │    │
│  │      "sentiment": "NEGATIVE",                           │    │
│  │      "aspects": [...],                                  │    │
│  │      "content_created_at": 1707577800                   │    │
│  │    }                                                    │    │
│  │  }                                                      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                         ↓                                       │
│  Step 4: Upsert to Qdrant                                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  → Qdrant                                               │    │
│  │    POST /collections/smap_analytics/points/upsert       │    │
│  │  ← Success                                              │    │
│  └─────────────────────────────────────────────────────────┘    │
│                         ↓                                       │
│  Step 5: Save Metadata                                          │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  → PostgreSQL                                           │    │
│  │    INSERT INTO knowledge.indexed_documents              │    │
│  │    (analytics_id, project_id, vector_id, indexed_at)   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                         ↓                                       │
│  Step 6: Invalidate Cache                                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  → Redis                                                │    │
│  │    DEL search:{project_id}:*                            │    │
│  │    (Invalidate all search caches for this project)     │    │
│  └─────────────────────────────────────────────────────────┘    │
│                         ↓                                       │
│  Step 7: Publish Event (Optional)                              │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  → Kafka                                                │    │
│  │    Topic: knowledge.indexed                             │    │
│  │    Event: DOCUMENT_INDEXED                              │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ↓ Response
┌─────────────────────────────────────────────────────────────────┐
│  {                                                              │
│    "success": true,                                             │
│    "vector_id": "analytics_001",                                │
│    "indexed_at": "2026-02-15T10:46:45Z"                         │
│  }                                                              │
└─────────────────────────────────────────────────────────────────┘
```

**Latency Breakdown:**
- Step 1 (Validation): 5ms
- Step 2 (Embedding): 800ms (or 5ms if cached)
- Step 3 (Prepare): 5ms
- Step 4 (Qdrant Upsert): 100ms
- Step 5 (Save Metadata): 20ms
- Step 6 (Cache Invalidation): 5ms
- Step 7 (Kafka): 5ms (async)

**Total:** ~940ms (or ~140ms with cache hit)

---

### Flow 3: Report Generation

```
┌─────────────────────────────────────────────────────────────────┐
│  USER: "Tạo báo cáo so sánh VinFast vs BYD"                     │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ↓ POST /api/v1/reports/generate
┌─────────────────────────────────────────────────────────────────┐
│  KNOWLEDGE SERVICE                                              │
│                                                                 │
│  Step 1: Create Report Record                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  → PostgreSQL                                           │    │
│  │    INSERT INTO knowledge.reports                        │    │
│  │    (id, campaign_id, status='PROCESSING')              │    │
│  │  ← report_id: "report_789"                              │    │
│  └─────────────────────────────────────────────────────────┘    │
│                         ↓                                       │
│  Step 2: Return Async Response                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  {                                                      │    │
│  │    "report_id": "report_789",                           │    │
│  │    "status": "PROCESSING",                              │    │
│  │    "estimated_time_seconds": 30                         │    │
│  │  }                                                      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                         ↓                                       │
│  Step 3: Background Job (Goroutine)                            │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  3.1. Get Campaign Projects                             │    │
│  │       → Project Service                                 │    │
│  │                                                         │    │
│  │  3.2. Search Qdrant for all relevant data              │    │
│  │       → Qdrant (multiple queries)                       │    │
│  │                                                         │    │
│  │  3.3. Aggregate statistics                              │    │
│  │       - Sentiment distribution                          │    │
│  │       - Aspect breakdown                                │    │
│  │       - Top mentions                                    │    │
│  │                                                         │    │
│  │  3.4. Generate report content                           │    │
│  │       → OpenAI API (GPT-4)                              │    │
│  │       Prompt: "Generate comparison report..."           │    │
│  │                                                         │    │
│  │  3.5. Convert to PDF                                    │    │
│  │       Library: wkhtmltopdf / chromedp                   │    │
│  │                                                         │    │
│  │  3.6. Upload to MinIO                                   │    │
│  │       → MinIO                                           │    │
│  │       PUT /smap-reports/reports/report_789.pdf          │    │
│  │       ← presigned_url (valid 7 days)                    │    │
│  │                                                         │    │
│  │  3.7. Update report record                              │    │
│  │       → PostgreSQL                                      │    │
│  │       UPDATE knowledge.reports                          │    │
│  │       SET status='COMPLETED', file_url=...              │    │
│  │                                                         │    │
│  │  3.8. Publish event                                     │    │
│  │       → Kafka                                           │    │
│  │       Topic: knowledge.reports                          │    │
│  │       Event: REPORT_COMPLETED                           │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Latency Breakdown:**
- Step 1 (Create Record): 20ms
- Step 2 (Return Response): 5ms
- Step 3.1 (Get Projects): 50ms
- Step 3.2 (Search Qdrant): 500ms
- Step 3.3 (Aggregate): 200ms
- Step 3.4 (LLM Generate): 15000ms
- Step 3.5 (Convert PDF): 3000ms
- Step 3.6 (Upload MinIO): 500ms
- Step 3.7 (Update DB): 20ms
- Step 3.8 (Kafka): 5ms

**Total Background:** ~19s

---

## DEPENDENCY INTERACTION MATRIX

| Operation | Qdrant | OpenAI | PostgreSQL | Redis | Project Svc | MinIO | Kafka |
|-----------|--------|--------|------------|-------|-------------|-------|-------|
| **Chat** | ✅ Search | ✅ Embed + LLM | ✅ Save history | ✅ Cache | ✅ Get projects | ❌ | ⭕ Event |
| **Search** | ✅ Search | ✅ Embed | ❌ | ✅ Cache | ✅ Get projects | ❌ | ⭕ Event |
| **Index** | ✅ Upsert | ✅ Embed | ✅ Save metadata | ✅ Cache | ❌ | ❌ | ⭕ Event |
| **Report** | ✅ Search | ✅ LLM | ✅ Save metadata | ❌ | ✅ Get projects | ✅ Upload | ⭕ Event |

**Legend:**
- ✅ Required
- ❌ Not used
- ⭕ Optional

---

## FAILURE SCENARIOS

### Scenario 1: Qdrant Down

**Impact:** ❌ CRITICAL - Service unavailable

**Affected Operations:**
- Chat: Cannot search → Cannot answer
- Search: Cannot search → No results
- Index: Cannot upsert → Data not indexed
- Report: Cannot search → Cannot generate

**Mitigation:**
- Health check returns "unhealthy"
- Return 503 Service Unavailable
- Queue requests for retry (optional)

---

### Scenario 2: OpenAI API Down

**Impact:** ❌ CRITICAL - Service unavailable

**Affected Operations:**
- Chat: Cannot generate answer (search still works)
- Index: Cannot generate embeddings → Cannot index

**Mitigation:**
- Retry with exponential backoff
- Fallback to cached embeddings (if available)
- Return error with retry suggestion

---

### Scenario 3: Redis Down

**Impact:** ⚠️ DEGRADED - Service works but slower

**Affected Operations:**
- Chat: No cache → Slower (5s → 6s)
- Search: No cache → Slower
- Rate limiting: Disabled

**Mitigation:**
- Continue without cache
- Log warning
- Monitor performance degradation

---

### Scenario 4: PostgreSQL Down

**Impact:** ⚠️ DEGRADED - Some features unavailable

**Affected Operations:**
- Chat: Works but no history saved
- Index: Works but no metadata tracking
- Report: Cannot save metadata

**Mitigation:**
- Continue core operations
- Queue writes for retry
- Return warning in response

---

### Scenario 5: Project Service Down

**Impact:** ❌ CRITICAL - Cannot resolve campaigns

**Affected Operations:**
- Chat: Cannot get campaign projects → Cannot filter
- Report: Cannot get campaign projects → Cannot generate

**Mitigation:**
- Cache campaign → project mappings in Redis
- Fallback to cache if Project Service down
- Return error if no cache available

---

### Scenario 6: MinIO Down

**Impact:** ⚠️ DEGRADED - Reports unavailable

**Affected Operations:**
- Report: Cannot upload → Generation fails
- Chat/Search: Not affected

**Mitigation:**
- Return error for report generation
- Core chat/search still works

---

## MONITORING CHECKLIST

### Health Checks

```bash
# Overall health
curl http://localhost:8080/health

# Detailed health with dependencies
curl http://localhost:8080/health/detailed
```

### Metrics to Monitor

**Latency:**
- `knowledge_chat_duration_seconds` (P50, P95, P99)
- `knowledge_search_duration_seconds`
- `knowledge_embedding_duration_seconds`
- `knowledge_llm_duration_seconds`

**Throughput:**
- `knowledge_requests_total{endpoint}`
- `knowledge_embeddings_generated_total`
- `knowledge_searches_total`

**Errors:**
- `knowledge_errors_total{type, dependency}`
- `knowledge_openai_errors_total`
- `knowledge_qdrant_errors_total`

**Cache:**
- `knowledge_cache_hits_total`
- `knowledge_cache_misses_total`
- `knowledge_cache_hit_rate` (target: > 60%)

**Dependencies:**
- `knowledge_dependency_up{dependency}` (0 or 1)
- `knowledge_dependency_latency_seconds{dependency}`

---

**Chi tiết đầy đủ:** Xem `KNOWLEDGE_SERVICE_DEPENDENCIES.md`
