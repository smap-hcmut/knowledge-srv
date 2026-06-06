# SMAP Knowledge Service — Agent Reference

Service-level deep dive for knowledge-srv codebase. Covers architecture, data flow, API shapes, internals, and known issues.

---

## 1. Service Shape

### Entry Points

| Component | File | Port | Role |
|-----------|------|------|------|
| **HTTP Server** | `cmd/server/main.go:65–252` | 8080 | REST API for chat, search, reports, internal indexing ops |
| **Kafka Consumer** | `internal/consumer/server.go:62–83` | — | Async Kafka ingestion (3 topics: batch, insights, digest) |

Single combined deployment (`cmd/server/main.go`): HTTP server + consumer run concurrently on the same process. No separate consumer pod in most deployments.

### Config Loading
- **Location**: `config/config.go` (struct) + `config/knowledge-config.yaml` (values)
- **Key Env Vars**: (set via env or YAML)
  - `VOYAGE_API_KEY` — Voyage AI embedding key
  - `LLM_PROVIDERS` / legacy `GEMINI_API_KEY` — Multi-provider LLM with fallback
  - `QDRANT_HOST:QDRANT_PORT` — Qdrant gRPC endpoint (6334)
  - `POSTGRES_HOST:POSTGRES_PORT` — PG metadata store
  - `REDIS_HOST:REDIS_PORT` — Cache & rate limit
  - `MINIO_ENDPOINT` — S3-compatible report storage
  - `KAFKA_BROKERS` — Kafka consumer broker list
  - `INTERNAL_KEY` — Auth for internal endpoints (`/internal/*`)
- **Multi-LLM Support**: `LLMConfig.Providers` (Gemini, OpenAI, DeepSeek, Qwen) with fallback chain

---

## 2. Indexing Pipeline

### Kafka Topics Consumed

| Topic | Consumer Group | Handler | Purpose |
|-------|----------------|---------|---------|
| `analytics.batch.completed` | `knowledge-indexing-batch` | `handleBatchCompleted` | Layer 3: Raw social posts (legacy MinIO + new direct payload) |
| `analytics.insights.published` | `knowledge-indexing-insights` | `handleInsightsPublished` | Layer 2: Aggregated insight cards (macro-level) |
| `analytics.report.digest` | `knowledge-indexing-digest` | `handleReportDigest` | Layer 1: Campaign-level digest summary |

**Message Envelope** (Type Definitions)
- `BatchCompletedMessage` — `internal/indexing/delivery/kafka/type.go:39–43`
  - `project_id`, `campaign_id`, `documents[]` (direct payload)
  - Each doc has: `identity` (UapID, platform), `content` (clean_text), `nlp` (sentiment, aspects), `business` (impact, relevance), `rag` (flag)
- `InsightsPublishedMessage` — `internal/indexing/delivery/kafka/type.go:130–143`
  - `project_id`, `campaign_id`, `run_id`, `insight_type`, `title`, `summary`, `confidence`, `should_index` flag
- `ReportDigestMessage` — `internal/indexing/delivery/kafka/type.go:145–159`
  - Top entities/topics/issues, total mentions, `should_index` flag

### shouldIndexInsight Gate (Known Tuning Point)
- **Location**: `internal/indexing/usecase/index_batch.go:246–267`
- **Current State**: PERMISSIVE fallback (lines 264–266)
  - OLD BUG: `rag=false` was rejecting all non-fallback-platform records → fixed
  - **NOW**: Accepts most records unless:
    1. Content < 10 chars → skip
    2. Business relevance score < 0.3 AND no signal (aspects/entities/high impact) AND not social platform → skip
    3. Otherwise → accept ("default_allow_fallback" gate)
- **Why Permissive**: Demo data is sparse; strict quality gates = silent data starvation
- **Levers**:
  - `MinContentLength = 10` — `internal/indexing/types.go:9`
  - `MinBusinessRelevanceScore = 0.3` — `internal/indexing/types.go:11`
  - `hasInsightSignal()` / `containsBusinessSignal()` — check aspects, entities, impact score, priority, keyword matches
  - `isSocialPlatformFallback()` — whitelist: TikTok, Facebook, Instagram, X, YouTube, Threads, Reddit

### Point ID Derivation (Previously Buggy)
- **OLD BUG**: Raw TikTok IDs used as UUIDs → collisions, schema violations
- **NOW FIXED**: Deterministic UUID v5 (SHA1) — `internal/indexing/usecase/index_batch.go:354–365`
  ```go
  analyticsID := deterministicInsightUUID("analytics", projectID, doc.Identity.UapID)
  sourceID := deterministicInsightUUID("source", projectID, rootID_or_postURL_or_...)
  pointID := analyticsID  // UUIDs always valid per RFC 4122
  ```
  Seed combines scope + project + raw ID → deterministic, idempotent, never collides

### Processing Flow
1. **Parse**: Kafka message → structs
2. **Validate**: Required fields present (UapID, clean_text, project_id)
3. **Pre-filter**: `rag` flag, platform fallback, content quality
4. **Quality Check**: `shouldIndexInsight()` gate
5. **Embed**: Voyage multilingual-2 (1024 dims) via embedding UseCase
6. **Upsert to Qdrant**: Point = (pointID, vector, payload)
7. **Track in PostgreSQL**: `indexed_documents` table + metrics
8. **Cache Invalidate**: Redis search cache bust on success

### Retry Path
- **Endpoint**: `POST /internal/index/retry` (internal auth required)
- **Logic**: `internal/indexing/usecase/retry_failed.go:12–67`
  - Find records with `status=FAILED` AND `retry_count < max_retries`
  - Mark as `PENDING` (re-queue), increment retry count
  - Resolve matching DLQ entry
  - Useful for transient Voyage/Qdrant/network failures

### Reconcile Path
- **Endpoint**: `POST /internal/index/reconcile` (internal auth required)
- **Logic**: `internal/indexing/usecase/reconcile.go:12–55`
  - Find records stuck in `PENDING` status for > X duration
  - Mark as `FAILED` with reason "PENDING timeout"
  - Prevents orphaned in-flight documents

---

## 3. Qdrant Integration

### Collection Naming
- **Per-project**: `proj_{project_id}` (e.g., `proj_proj_123abc`) — stores raw posts + insights
- **Macro**: `macro_insights` — insight cards (Layer 2)
- **Macro**: `macro_insights` again (same) — digests (Layer 1)

Vector size: **1024 dimensions** (Voyage multilingual-2) — `internal/indexing/usecase/constants.go:3`

### Payload Schema (Qdrant Point)
Example from `internal/indexing/usecase/payload_mapper.go:82–122`:
```json
{
  "analytics_id": "uuid",
  "project_id": "uuid",
  "campaign_id": "uuid",
  "qdrant_point_id": "uuid",
  "platform": "tiktok|facebook|...",
  "content": "full post text",
  "sentiment_label": "positive|negative|neutral",
  "sentiment_score": 0.85,
  "aspects": ["delivery cost", "driver quality"],
  "entities": ["AhaMove", "competitor"],
  "impact_score": 0.67,
  "relevance_score": 0.45,
  "priority": "HIGH|MEDIUM|LOW",
  "likes": 42,
  "comments": 8,
  "shares": 2,
  "views": 1200,
  "published_at": "2026-01-15T10:30:00Z",
  "url": "post URL",
  "author": "username",
  "content_summary": "short summary",
  "context_summary": "enriched context from analytics"
}
```
Payload is **strictly schema-free** (map[string]interface{}) in Qdrant — no enforcement at storage level. **Pitfall**: If analysis-srv changes payload format, must coordinate here.

### Point ID Handling
- **UUID IDs** (modern approach): Stored as Qdrant UUID field directly (RFC 4122 compliant)
- **Non-UUID IDs** (legacy fallback): Hash to numeric uint64 via MD5 → `generateHashNumber()` — `pkg/qdrant/qdrant.go:198–201`
- **Detection**: `isValidUUID()` regex match — `pkg/qdrant/qdrant.go:18, 154–157`
- **Batch Size**: No hard limit per upsert, but code batches at 10 concurrent goroutines (`MaxConcurrency = 10`) — `internal/indexing/types.go:8`

### Filter Expansion (Qdrant Conditions)
Search filters apply string matching with case variants — `internal/search/usecase/filters.go:21–40`
- **Platforms**: `expandFilterKeywordVariants(["tiktok"])` → ["tiktok", "TikTok", "TIKTOK"] for FieldCondition.match
- **Sentiments**: Multi-condition OR (any of positive/negative/neutral match)
- **Aspects/Entities**: Keyword IN conditions
- **Date Range**: Timestamp GTE/LTE conditions

---

## 4. Voyage Embeddings

### Client Setup
- **Endpoint**: `https://api.voyageai.com/v1/embeddings`
- **Model**: `voyage-multilingual-2`
- **Batch Size**: Handled per-request in `Generate` (single text) / `GenerateMany` (multiple) — `internal/embedding/usecase/generate.go:11–47`
- **Timeout**: Context-based (caller sets timeout)
- **Caching**: Redis layer via embedding repository (SHA256 hash of text = key) — `internal/embedding/usecase/generate.go:17–24`
- **Error Handling**:
  - Network/API error → return `ErrEmbeddingFailed` (fatal, triggers document FAILED status)
  - No vector returned → `ErrNoVectorReturned` (fatal)
  - Cache save error → logged as warning (non-fatal, embedding succeeded)

---

## 5. Search API

### Endpoint Shape
`POST /api/v1/knowledge/search`

**Request DTO** — `internal/search/types.go`:
```go
SearchInput {
  CampaignID string          // Required
  Query      string          // Required: semantic search query
  Limit      int             // Optional: max results (default 20, max 50)
  MinScore   float64         // Optional: relevance threshold (default 0.2)
  Filters    SearchFilters   // Optional
}

SearchFilters {
  Sentiments []string  // ["positive", "negative", "neutral"]
  Aspects    []string  // ["delivery cost", "driver"]
  Platforms  []string  // ["tiktok", "facebook"]
  DateFrom   *time.Time
  DateTo     *time.Time
  RiskLevels []string
}
```

**Response DTO**:
```go
SearchOutput {
  Results           []SearchResult      // Ranked, de-duped, scored
  TotalFound        int
  Aggregations      SearchAggregations  // Sentiment/platform/aspect counts
  NoRelevantContext bool                // true = search returned 0 useful results
  CacheHit          bool
  ProcessingTimeMs  int64
}

SearchResult {
  ID            string
  Score         float64                 // Semantic relevance [0, 1]
  Content       string
  Metadata      map[string]interface{}  // Full Qdrant payload
  EngagementScore float64               // Derived from likes/comments/shares/views
  OverallSentiment string
}
```

### Search Flow (internal/search/usecase/search.go:24–166)
1. **Cache Check**: Key = hash(campaign_id + query + filters) — 3-layer cache
2. **Resolve Campaign**: Get project_ids for campaign (also cached)
3. **Enrich Query**: Prepend campaign name for better semantic matching
4. **Embed Query**: Voyage vector
5. **Build Filter**: Qdrant conditions for sentiments, aspects, platforms, date range
6. **Search Multi-Collections**: Query each `proj_{project_id}` collection in parallel
7. **Score Filter**: Keep results > minScore (default 0.2)
8. **Dedupe**: Remove repeated snapshots of same logical post (same UapID)
9. **Rank**: Custom scoring function (semantic score + business relevance + engagement + sentiment + content length)
10. **Limit**: Apply `limit` cap
11. **Cache Result**: Only if meaningful (not zero results, not error)
12. **Return**: Aggregations computed from final results

### Rankings
Scoring formula — `internal/search/usecase/search.go:186–200`:
```
rank_score = semantic_score * 10 
           + business_relevance * 2.2
           + log10(engagement_score + 1) * 0.45
           + min(content_length / 260, 1) * 0.25
           + sentiment_boost [0.20 for negative, 0.10 for positive, 0 for neutral]
```

### Pagination
None — fixed limit (max 50). Client must narrow query/filters for pagination effect.

---

## 6. Chat API

### Endpoint Shape
`POST /api/v1/knowledge/chat`

**Request DTO** — `internal/chat/types.go`:
```go
ChatInput {
  Message         string            // Required: user question
  CampaignID      string            // Required
  ConversationID  string            // Optional: existing conversation or new
  Filters         ChatFilters       // Optional: sentiment, platforms, date range
}
```

**Response DTO**:
```go
ChatOutput {
  ConversationID string            // New or existing
  Answer         string            // LLM-generated answer
  Citations      []Citation        // Evidence URLs/IDs from search results
  Suggestions    []string          // 3–5 follow-up questions
  SearchMetadata SearchMeta        // How many docs searched, used, timing
  QueryIntent    string            // "STRUCTURED" or "NARRATIVE"
  Backend        string            // "local-router" | "Qdrant" | "NotebookLM"
}

Citation {
  SourceID  string  // Point ID
  URL       string  // Post URL
  Platform  string
  Content   string  // Snippet
}
```

### RAG Flow (internal/chat/usecase/chat.go:22–215)
1. **Classify Intent**: Rule-based (not LLM) — `ClassifyIntent(message)`
   - **STRUCTURED**: "How many mentions?", "Top 10", "Filter by X" → sync (200 OK), limited search
   - **NARRATIVE**: "Analyze trends", "Insights?", "Compare" → would be async (202 Accepted) but fallback to sync
2. **Small Talk Bypass**: Local hard-coded response (no search/LLM)
3. **Analytics First** (conditional): For structured queries, try analytics-srv snapshot first (2s timeout)
4. **Search RAG**: Query Qdrant via search UseCase (limit=15 for narrative, 20 for structured, minScore=0.2)
5. **Analytics Fallback**: If search returns 0 results, try analytics snapshot (3s timeout)
6. **Build Prompt**: Combine message + top K search results + conversation history + analytics snapshot (optional)
7. **Call LLM** (60s timeout, concurrency-limited to 5 max across chat + report)
8. **Extract Citations**: URLs from search results ranked >= threshold
9. **Generate Suggestions**: Rule-based + LLM (if time permits)
10. **Persist**: User message + assistant answer + filters + search metadata → PostgreSQL `conversations` + `messages` tables

### Conversation Persistence
- **Table**: `knowledge.conversations` (campaign_id, user_id, title, status, message_count)
- **Messages**: `knowledge.messages` (conversation_id, role [user|assistant], content, filters_used, search_metadata, created_at)
- **History**: Loaded on continue (limit last 50 messages) for context

### Fallback Strategy
1. If search empty → try analytics snapshot (campaign KPIs, sentiment, platforms, top posts)
2. If analytics also empty → respond "not enough data, try narrower query"
3. Never hallucinate: Always require search hit OR analytics snapshot

---

## 7. Report API

### Endpoints
| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/api/v1/knowledge/reports/generate` | Request report (async, returns 202 + job_id) |
| `GET` | `/api/v1/knowledge/reports/:id` | Poll status (PROCESSING/COMPLETED/FAILED) |
| `GET` | `/api/v1/knowledge/reports/:id/download` | Get file from MinIO (markdown or PDF) |

**Generate Request**:
```go
GenerateInput {
  CampaignID string       // Required
  ReportType string       // "SUMMARY" | "COMPARISON" | "TREND" | "ASPECT_DEEP_DIVE"
  Filters {
    Sentiments    []string
    Aspects       []string
    Platforms     []string
    DateFrom      *time.Time
    DateTo        *time.Time
    Sections      []string  // Which sections to include
    CompetitorURLs []string // For comparison reports
  }
}
```

### Generator Pipeline (internal/report/usecase/generator.go:17–153)
1. **Aggregate**: Search for relevant docs (same as RAG but higher limit=50+)
2. **Evidence Pack**: Select 10–15 representative docs (balanced by platform/sentiment/topic)
3. **Analytics Summary**: Load KPI snapshot from analytics-srv (engagement, sentiment, top posts)
4. **Build Prompt**: Structured markdown with evidence pack IDs, analytics summary, section requests
5. **LLM Generate** (4 min timeout): One coherent business narrative
6. **Normalize Markdown**: Clean up LLM formatting
7. **Compile**: Assemble final markdown (header + sections + evidence citations)
8. **Upload to MinIO**: `reports/{report_id}.md` (or .pdf if converted)
9. **Update Status**: Mark COMPLETED, save file_url, file_size_bytes, generation_time_ms, sections_count

### Data Sources
- **Indexed Docs**: Via search UseCase (semantic + filter-based)
- **Analytics**: Campaign-level KPIs (total mentions, sentiment distribution, platform stats, top posts)
- **Template**: Business-grade report structure (not hardcoded slides, free-form LLM generation)

---

## 8. Content Quality Scoring

### Location & Purpose
`internal/contentquality/quality.go` — Filters low-value marketing spam from RAG context

### Filter Criteria
Applied in `isLowValueMarketingContent()` — returns true to skip:
- **Empty or hashtag-only**: Content is just "#tag #tag"
- **Marketing Noise Patterns**: 150+ hard-coded low-value strings (AhaMoveCareer, "workshop nội bộ", "tuyển dụng", "minigame", "rinh quà", "collshp.com", etc.)
- **Contact Farming**: Phone number + contact signal (Zalo, ib, "liên hệ", etc.)
- **Off-Topic**: e.g., "call of duty" (COD game, not delivery COD)

### When Applied
During search ranking — `internal/search/usecase/search.go:168–184`, results are marked as "not useful" if:
1. Empty content
2. Low-value marketing
3. Very low business relevance + low semantic score (biz < 0.30 && score < 0.68)

**Note**: Not a hard reject during indexing — documents are indexed, but filtered in search/chat results.

---

## 9. Analytics Client

### Endpoints Called
`pkg/analytics/client.go:59–85`

**Snapshot** — GET `/api/{campaign_id}/snapshot`
- Returns: KPIs, platform stats, sentiment donut, top keywords, top posts
- Used for: Chat fallback context, report context, analytics-first gate
- Timeout: Config-based (default 12s)
- **Cache**: 45s in-memory (immutable within window)

### Retry & Fallback
- If analytics-srv is slow/down → chat continues with search-only RAG
- Non-fatal: Snapshot load failure → use search results alone
- **Important**: Chat can succeed with zero analytics data (search+LLM sufficient)

---

## 10. Persistence

### PostgreSQL Schema
**Location**: `migrations/` directory (10 migration files)

| Table | Purpose | Key Columns |
|-------|---------|-------------|
| `indexed_documents` | Indexing tracking | `analytics_id` (unique), `project_id`, `qdrant_point_id`, `status` (PENDING/INDEXED/FAILED), `retry_count`, `content_hash`, embedding/upsert timing |
| `conversations` | Chat sessions | `id`, `campaign_id`, `user_id`, `title`, `status` (ACTIVE/ARCHIVED), `message_count`, `last_message_at` |
| `messages` | Chat history | `conversation_id`, `role` (user/assistant), `content`, `filters_used` (JSON), `search_metadata` (JSON) |
| `reports` | Report generation tracking | `id`, `campaign_id`, `user_id`, `report_type`, `status` (PROCESSING/COMPLETED/FAILED), `file_url`, `file_size_bytes`, `total_docs_analyzed`, `sections_count`, `generation_time_ms` |
| `indexing_dlq` | Dead-letter queue | `analytics_id`, `error_type`, `error_message`, `status` (NEW/RESOLVED) |
| `indexing_error_summary` | Error aggregation | Stats by project/batch/error type |
| `indexing_stats_by_project` | Monitoring view | COUNT by status, avg timing per project |
| `indexing_stats_by_batch` | Monitoring view | COUNT by status per batch |

**Schema Namespace**: `schema_knowledge` (not public) — Important for multi-tenant isolation

### Migrations
- Run in order (001 → 010)
- Each is idempotent (IF NOT EXISTS)
- Latest: `010_drop_notebook_tables.sql` (NotebookLM integration deprecated)

---

## 11. Statistics API

### Endpoint
`GET /internal/index/statistics/:project_id`

**Response** — `internal/indexing/usecase/get_statistics.go:10–25`:
```go
StatisticOutput {
  ProjectID      string
  TotalIndexed   int       // status = INDEXED
  TotalFailed    int       // status = FAILED
  TotalPending   int       // status = PENDING
  LastIndexedAt  *time.Time
  AvgIndexTimeMs float64   // Average total_time_ms
}
```

**Use Case**: Monitoring dashboard, debugging indexing stalls, alerting on error rate

---

## 12. Known Bugs & Fragile Spots

### Resolved Issues
1. **Raw TikTok IDs as UUIDs** ✅ FIXED — Now deterministic SHA1 UUID (idempotent)
2. **shouldIndexInsight gate rejecting all `rag=false`** ✅ FIXED — Added platform fallback + default allow
3. **Qdrant UUID vs numeric ID handling** ✅ FIXED — Type-safe dispatch with regex validation

### Current Fragile Areas

| Area | Risk | Mitigation |
|------|------|-----------|
| **Payload schema drift** | analysis-srv changes format → payload mismatch | Always coordinate schema changes between analysis-srv & knowledge-srv |
| **Qdrant collection naming** | Hardcoded `proj_{project_id}` — collisions if project IDs collide | Verify project_id uniqueness upstream (project-srv responsibility) |
| **Embedding cache expiry** | Redis TTL not set on embedding cache → stale vectors | Embedding caching has NO TTL (infinite), only invalidated on manual clear |
| **Analytics snapshot timeout** | 2–3s fallback might return partial data | Load analytics early in chat flow to avoid timeout race |
| **LLM rate limiting** | Semaphore maxes at 5 concurrent calls (chat + report share pool) | Can queue up; consider separate pools if concurrency increases |
| **Search cache poisoning** | Empty search results ARE NOT cached (intentional), but transient failures look like no data | Search returns `NoRelevantContext=true` only if fetch succeeded but returned 0 useful docs |
| **Report generation memory** | Full evidence pack + full LLM output loaded in memory (not streaming) | Reports are single-process; huge campaigns might OOM (mitigate: cap evidence to 15 docs) |

### Debug Levers
- **Logger Level**: Set `LOG_LEVEL` env var (debug, info, warn, error)
- **shouldIndexInsight Override**: No env override; modify `default_allow_fallback` in `index_batch.go:266` to return `false` for stricter gating
- **Search Cache Bypass**: Pass `X-Bypass-Cache: true` header (if implemented in HTTP handler)
- **Analytics Fallback Disable**: Set `router.notebook_fallback_enabled=false` in config (but this only affects NotebookLM, not analytics snapshot)

---

## 13. Dev & Test Commands

### Build
```bash
# From knowledge-srv/ root:
make swagger        # Generate Swagger docs from code annotations
make models         # Run sqlboiler to gen models from migrations
make run            # Run API + consumer locally
```

### Tests
```bash
make test           # Unit tests (if configured)
```

### Docker
```bash
docker build -t knowledge-srv:latest -f cmd/server/Dockerfile .
docker run -p 8080:8080 -v $(pwd)/config:/app/config knowledge-srv:latest
```

### Kubernetes
```bash
kubectl apply -f manifests/configmap.yaml
kubectl apply -f manifests/secret.yaml
kubectl apply -f manifests/
kubectl logs -l app=knowledge-api --tail=100 -f
```

---

## 14. Pitfalls for Fresh Agents

### Do NOT:
1. **Tighten `shouldIndexInsight` without E2E testing** — Demo data is sparse; strict gates = silent data loss
2. **Change Qdrant payload schema without coordination** — analysis-srv producer must be aware; point IDs might change
3. **Assume collection names are stable** — `proj_{project_id}` is by design, but project_id format MUST be UUID (else ID collision risk)
4. **Rely on Embedding Redis cache expiry** — It's infinite TTL; clear manually if semantics change
5. **Forget to invalidate search cache on index success** — Stale results served to chat
6. **Use numeric point IDs for new records** — Always generate proper UUIDs (deterministic SHA1 or random v4), fallback hash is for compat only
7. **Send LLM timeout < 60s for chat** — Concurrent calls + API latency = stalls; 60s is tuned baseline
8. **Skip coordinator check for payload schema changes** — Qdrant is schema-free, but downstream readers (search, report) depend on specific field names

### Key Dependencies:
- **With analysis-srv**: Payload schema, batch/insight/digest message formats, analytics snapshot schema
- **With project-srv**: Campaign → project_id resolution (search query enrichment)
- **With Voyage**: Embedding API availability (fallback: retry + circuit break?)
- **With Qdrant**: Collection naming, vector size (1024), point ID format
- **With Gemini/LLM**: Multi-provider fallback behavior, timeout handling

---

## 15. Entry Points Summary

| Use Case | HTTP Endpoint | Internal Call | Kafka Topic |
|----------|---------------|---------------|-------------|
| Search posts | `POST /api/v1/knowledge/search` | `searchUC.Search()` | — |
| Chat (RAG) | `POST /api/v1/knowledge/chat` | `chatUC.Chat()` | — |
| Generate report | `POST /api/v1/knowledge/reports/generate` | `reportUC.Generate()` (async) | — |
| Manual index | `POST /internal/index` | `indexingUC.Index()` | — |
| Retry failed | `POST /internal/index/retry` | `indexingUC.RetryFailed()` | — |
| Reconcile PENDING | `POST /internal/index/reconcile` | `indexingUC.Reconcile()` | — |
| Get stats | `GET /internal/index/statistics/:project_id` | `indexingUC.GetStatistics()` | — |
| — | — | — | `analytics.batch.completed` |
| — | — | — | `analytics.insights.published` |
| — | — | — | `analytics.report.digest` |

---

**Last Updated**: 2026-06-06  
**Version**: 1.0
