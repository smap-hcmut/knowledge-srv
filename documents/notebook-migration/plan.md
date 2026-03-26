# Knowledge Service — Migration Execution Plan

> Dựa trên `MIGRATION_PROPOSAL.md` v2.0 | Maestro prod: `https://maestro.tantai.dev/maestro`
> Ngày tạo: 2026-03-24 | Cập nhật: 2026-03-24 (refactor theo convention)

---

## Convention phân tích

### `pkg/` — Thin infrastructure clients (no business logic)

```text
pkg/voyage/       → interface.go (IVoyage), type.go, constant.go, voyage.go
pkg/qdrant/       → interface.go (IQdrant), types.go, errors.go, qdrant.go
pkg/projectsrv/   → interface.go (IProject), type.go, projectsrv.go
```

Pattern: flat package, interface.go định nghĩa interface, impl cùng package, không sub-packages.

### `internal/<domain>/` — Business domains (clean architecture)

```text
internal/chat/      → interface.go, types.go, usecase/, delivery/http/, repository/postgre/
internal/indexing/  → interface.go, types.go, usecase/, delivery/kafka/, repository/postgre/
internal/embedding/ → usecase/, repository/redis/
internal/point/     → usecase/, repository/qdrant/
```

Pattern: flat domain dưới internal/, với usecase/, delivery/, repository/ sub-packages.

---

## Kiến trúc migration — Theo đúng convention

```text
pkg/
└── maestro/                     # Thin HTTP client (giống pkg/voyage/)
    ├── interface.go             # IMaestro — pure I/O, no state
    ├── types.go                 # DTOs khớp OpenAPI 0.6.0
    ├── maestro.go               # HTTP calls only
    ├── errors.go
    └── constant.go

internal/
├── notebook/                    # Business domain (giống internal/chat/)
│   ├── interface.go
│   ├── types.go
│   ├── errors.go
│   ├── usecase/
│   │   ├── new.go
│   │   ├── session.go           # Session lifecycle (business logic)
│   │   ├── ensure.go            # EnsureNotebook
│   │   ├── sync.go              # SyncPart + pollJobUntilDone
│   │   ├── query.go             # Phase 3: QueryNotebook
│   │   ├── retry.go
│   │   └── webhook.go
│   ├── delivery/http/
│   │   ├── new.go
│   │   ├── handlers.go          # Webhook + async polling
│   │   └── routes.go
│   └── repository/postgre/
│       ├── new.go
│       ├── campaign.go
│       ├── source.go
│       ├── session.go
│       └── chat_job.go          # Phase 3
│
└── transform/
    ├── interface.go
    ├── types.go
    └── usecase/
        ├── new.go
        ├── build_markdown.go
        └── batch_builder.go
```

So sánh với proposal cũ: xem chi tiết trong `MIGRATION_PLAN_DETAILS.md`.

---

## Tổng quan 4 Phases

Chi tiết từng phase xem `MIGRATION_PLAN_DETAILS.md`.

### Phase 1 — Maestro Client + Transform (Tuần 1-2)

Không break flow hiện tại.

**Files mới:**

- `migrations/008_create_notebook_tables.sql` — 3 tables
- `pkg/maestro/` — 5 files (interface, types, impl, errors, constant)
- `internal/transform/` — 5 files (interface, types, usecase/new, build_markdown, batch_builder)

**Files sửa:**

- `config/config.go` — thêm MaestroConfig, NotebookConfig, RouterConfig
- `config/knowledge-config.yaml` — thêm maestro/notebook/router sections
- `internal/indexing/delivery/kafka/type.go` — thêm CampaignID
- `internal/indexing/types.go` — thêm CampaignID vào IndexInput
- `internal/indexing/delivery/kafka/consumer/presenters.go` — map CampaignID
- `cmd/api/main.go` — init maestroClient

**BLOCKER**: analysis-srv cần thêm campaign_id vào Kafka message.

### Phase 2 — Notebook Sync + Webhook (Tuần 3-4)

Feature flag notebook.enabled=false mặc định.

**Files mới:**

- `internal/notebook/` — 14 files (interface, types, errors, usecase/*, delivery/http/*, repository/postgre/*)
- `internal/httpserver/domain_notebook.go`

**Files sửa:**

- `internal/consumer/handler.go` — inject notebookUC, trigger sync
- `internal/consumer/new.go` — thêm maestro + notebook deps
- `internal/httpserver/handler.go` — register webhook route
- `internal/httpserver/new.go` — accept notebook deps
- `internal/indexing/delivery/kafka/consumer/workers.go` — trigger sync after index

### Phase 3 — Async Chat + Query Router (Tuần 5-6)

**Files mới:**

- `migrations/009_create_notebook_chat_jobs.sql`
- `internal/chat/usecase/router.go` — rule-based intent classifier
- `internal/chat/delivery/http/handlers_async.go` — GET /chat/jobs/{id}

**Files sửa:**

- `internal/chat/usecase/chat.go` — route NARRATIVE→notebook, STRUCTURED→qdrant
- `internal/chat/types.go` — thêm Backend, QueryIntent, ChatJobID, IsAsync
- `internal/notebook/usecase/query.go` — implement QueryNotebook
- `internal/notebook/repository/postgre/chat_job.go` — implement

### Phase 4 — RAG Improvement (Song song, độc lập)

- `internal/search/types.go` — MinScore 0.65→0.72, MaxResults 10→7
- `internal/chat/types.go` — MaxDocContentLen 500→800

---

## Dependency Map

```text
[analysis-srv: thêm campaign_id]  ← BLOCKER
       │
       ▼
Phase 1 (Tuần 1–2) ─────────────► Phase 4 (Song song)
  pkg/maestro/                      RAG Improvement
  internal/transform/
       │
       ▼
Phase 2 (Tuần 3–4)
  internal/notebook/
  Webhook + Feature Flag
       │
       ▼
Phase 3 (Tuần 5–6)
  Async Chat + Router
```

---

## Risks and Mitigation

| Risk | Severity | Mitigation |
| ---- | -------- | ---------- |
| Maestro UI automation break | HIGH | Feature flag notebook.enabled=false. Qdrant always fallback. |
| Chat latency 10-60s | HIGH | Async pattern + polling. Timeout 45s → fallback Qdrant. |
| Session pool 503 | MEDIUM | Retry + backoff. Non-blocking sync. |
| campaign_id missing | MEDIUM | Fallback: resolve via pkg/projectsrv (cached Redis 1h). |
| Webhook delivery failure | LOW | Fallback poll GET /jobs/{jobId} after timeout. |
