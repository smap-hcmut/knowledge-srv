# Migration Plan — Chi tiết kỹ thuật

Tham chiếu `MIGRATION_PLAN.md` cho tổng quan.

---

## Phase 1 Detail — pkg/maestro/ + internal/transform/

### pkg/maestro/interface.go

```go
package maestro

import "context"

type IMaestro interface {
    CreateSession(ctx context.Context, req CreateSessionReq) (SessionData, error)
    GetSession(ctx context.Context, sessionID string) (SessionData, error)
    DeleteSession(ctx context.Context, sessionID string) error
    CreateNotebook(ctx context.Context, sessionID string, req CreateNotebookReq) (JobEnqueued, error)
    ListNotebooks(ctx context.Context, sessionID string) ([]NotebookData, error)
    UploadSources(ctx context.Context, sessionID, notebookID string, req UploadSourcesReq) (JobEnqueued, error)
    ChatNotebook(ctx context.Context, sessionID, notebookID string, req ChatNotebookReq) (JobEnqueued, error)
    GetJob(ctx context.Context, jobID string) (JobData, error)
    SubmitPipeline(ctx context.Context, sessionID string, req PipelineReq) (PipelineData, error)
}
```

Pattern giống `pkg/voyage/interface.go` (IVoyage) — pure I/O, no state.

### pkg/maestro/maestro.go

```go
package maestro

type maestroImpl struct {
    baseURL    string
    apiKey     string
    httpClient pkghttp.Client // github.com/smap-hcmut/shared-libs/go/httpclient
}

func NewMaestro(cfg MaestroConfig) (IMaestro, error) {
    // Validate cfg.BaseURL, cfg.APIKey
    // Return &maestroImpl{...}
}
```

Pattern giống `pkg/voyage/interface.go` NewVoyage().

### pkg/maestro/constant.go

```go
package maestro

const (
    PathSessions  = "/notebooklm/sessions"
    PathNotebooks = "/notebooklm/notebooks"
    PathJobs      = "/notebooklm/jobs"
    PathPipelines = "/notebooklm/pipelines"

    StatusReady      = "ready"
    StatusBusy       = "busy"
    StatusTearDown   = "tearingDown"
    JobQueued        = "queued"
    JobProcessing    = "processing"
    JobCompleted     = "completed"
    JobFailed        = "failed"
)
```

### pkg/maestro/types.go

Request/Response DTOs khớp Maestro OpenAPI 0.6.0. Xem `MIGRATION_PROPOSAL.md` Section 4.

### pkg/maestro/errors.go

```go
var (
    ErrSessionNotFound = errors.New("maestro: session not found")
    ErrJobNotFound     = errors.New("maestro: job not found")
    ErrJobFailed       = errors.New("maestro: job failed")
    ErrUnauthorized    = errors.New("maestro: unauthorized")
    ErrSessionBusy     = errors.New("maestro: session busy")
    ErrRateLimited     = errors.New("maestro: rate limited")
)
```

---

### internal/transform/interface.go

```go
package transform

type UseCase interface {
    BuildParts(input TransformInput) ([]MarkdownPart, error)
}
```

### internal/transform/types.go

```go
type TransformInput struct {
    CampaignID   string
    CampaignName string
    Posts        []AnalyticsPostLite
}

type MarkdownPart struct {
    Title       string // "SMAP | Campaign X | 2026-W12 | Part 1"
    Content     string
    WeekLabel   string // "2026-W12"
    PartNum     int
    PostCount   int
    ContentHash string // SHA256 for dedup
}
```

### Kafka message change

`internal/indexing/delivery/kafka/type.go` — thêm CampaignID (optional, backward compat):

```go
type BatchCompletedMessage struct {
    BatchID     string    `json:"batch_id"`
    ProjectID   string    `json:"project_id"`
    CampaignID  string    `json:"campaign_id"`  // NEW
    FileURL     string    `json:"file_url"`
    RecordCount int       `json:"record_count"`
    CompletedAt time.Time `json:"completed_at"`
}
```

Propagate CampaignID qua IndexInput + presenters.go.

---

## Phase 2 Detail — internal/notebook/

### internal/notebook/interface.go

```go
package notebook

import "context"

type UseCase interface {
    StartSessionLoop(ctx context.Context) error
    StopSessionLoop(ctx context.Context) error
    EnsureNotebook(ctx context.Context, campaignID, periodLabel string) (NotebookInfo, error)
    SyncPart(ctx context.Context, input SyncPartInput) error
    RetryFailed(ctx context.Context) (RetryOutput, error)
    HandleWebhook(ctx context.Context, payload WebhookPayload) error
}
```

### internal/notebook/usecase/new.go

```go
type implUseCase struct {
    maestro      maestro.IMaestro    // pkg/maestro/ client
    campaignRepo repository.CampaignRepo
    sourceRepo   repository.SourceRepo
    sessionRepo  repository.SessionRepo
    transformUC  transform.UseCase
    cfg          Config
    l            log.Logger
    mu           sync.RWMutex
    activeSession string
    healthCancel  context.CancelFunc
}
```

Inject pattern giống `internal/chat/usecase/new.go`.

### Key flows

**ensureSession** (`usecase/session.go`):
1. RLock check activeSession
2. If empty: createAndPersistSession via maestro.CreateSession
3. Background healthcheck loop mỗi 60s via maestro.GetSession
4. If 404/tearingDown → recreate + update maestro_sessions table

**EnsureNotebook** (`usecase/ensure.go`):
1. Query notebook_campaigns WHERE campaign_id + period_label
2. If found → return
3. Else: ensureSession → maestro.CreateNotebook → pollJobUntilDone → INSERT

**SyncPart** (`usecase/sync.go`):
1. Check notebook_sources (SYNCED → skip idempotent)
2. EnsureNotebook
3. maestro.UploadSources with webhookUrl from config
4. Update notebook_sources.maestro_job_id, status=UPLOADING
5. Webhook callback will async update to SYNCED/FAILED

**pollJobUntilDone** (private method trong `usecase/sync.go`):
1. Loop: maestro.GetJob(jobID)
2. Linear 2s interval, max N attempts from config
3. Return result on completed, error on failed/timeout

**HandleWebhook** (`usecase/webhook.go`):
1. Route by action: upload_sources → update notebook_sources
2. create_notebook → update notebook_campaigns
3. chat → update notebook_chat_jobs (Phase 3)

### Webhook delivery endpoint

`internal/notebook/delivery/http/handlers.go`:
- POST /internal/notebook/callback
- Verify HMAC-SHA256(body, webhookSecret) == X-Maestro-Signature
- Parse payload → notebookUC.HandleWebhook

Register trong `internal/httpserver/handler.go` registerDomainRoutes.

### Consumer integration

`internal/indexing/delivery/kafka/consumer/workers.go` — sau Index thành công:

```go
if c.notebookEnabled && input.CampaignID != "" {
    go func() {
        // 1. transform.BuildParts(campaignID, records)
        // 2. notebook.SyncPart(campaignID, parts)
    }()
}
```

Non-blocking, behind feature flag.

---

## Phase 3 Detail — Async Chat + Router

### DB Migration — `migrations/009_create_notebook_chat_jobs.sql`

```sql
CREATE TABLE notebook_chat_jobs (
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
```

### Query Router — `internal/chat/usecase/router.go`

Rule-based v1: string matching cho tiếng Việt signals.
- NARRATIVE signals: "xu hướng", "đánh giá", "tổng quan", "phân tích"...
- STRUCTURED signals: "bao nhiêu", "thống kê", "top", "so sánh"...
- Default: STRUCTURED (safe fallback to Qdrant)

### Chat flow refactor — `internal/chat/usecase/chat.go`

```text
Chat(ctx, sc, input):
  1. Validate + load conversation (giữ nguyên)
  2. route = ClassifyIntent(input.Message)
  3. if route.UseNotebook && config.Notebook.Enabled && notebookAvailable:
       → return 202 async with chat_job_id
  4. else:
       → existing Qdrant flow (return 200 sync)
```

Thêm fields vào ChatOutput: Backend, QueryIntent, ChatJobID, IsAsync.

### Async endpoint

GET /api/v1/knowledge/chat/jobs/{chat_job_id}
- 202: pending/processing
- 200: completed answer
- 410: expired (>10 min)

Fallback: timeout 45s → auto-run Qdrant path, set fallback_used=true.

---

## Phase 4 — RAG Improvement (song song)

- `internal/search/types.go`: MinScore 0.65→0.72, MaxResults 10→7
- `internal/chat/types.go`: MaxDocContentLen 500→800
