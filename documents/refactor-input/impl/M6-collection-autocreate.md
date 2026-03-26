# M6: Collection Auto-Creation + Final Wiring

**Prerequisite**: M1–M5 phải done.

**Goal**: Auto-create Qdrant collections khi cần, verify end-to-end flow, verify graceful shutdown.

---

## Files cần thay đổi/tạo mới

### 1. `internal/point/interface.go`

**REPLACE toàn bộ file** — thêm `EnsureCollection`:

```go
package point

import (
	"context"
	"knowledge-srv/internal/model"
)

//go:generate mockery --name UseCase
type UseCase interface {
	Search(ctx context.Context, input SearchInput) ([]SearchOutput, error)
	Upsert(ctx context.Context, input UpsertInput) error
	Count(ctx context.Context, input CountInput) (uint64, error)
	Delete(ctx context.Context, input DeleteInput) error
	Scroll(ctx context.Context, input ScrollInput) ([]model.Point, error)
	Facet(ctx context.Context, input FacetInput) ([]FacetOutput, error)

	// EnsureCollection tạo collection nếu chưa tồn tại.
	// vectorSize phải match embedding model dimension (Voyage AI = 1024).
	EnsureCollection(ctx context.Context, name string, vectorSize uint64) error
}
```

---

### 2. `internal/point/repository/interface.go`

**REPLACE toàn bộ file** — thêm `EnsureCollection`:

```go
package repository

import (
	"context"

	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
)

//go:generate mockery --name QdrantRepository
type QdrantRepository interface {
	Search(ctx context.Context, opt SearchOptions) ([]point.SearchOutput, error)
	Upsert(ctx context.Context, opt UpsertOptions) error
	Count(ctx context.Context, opt CountOptions) (uint64, error)
	Delete(ctx context.Context, opt DeleteOptions) error
	Scroll(ctx context.Context, opt ScrollOptions) ([]model.Point, error)
	Facet(ctx context.Context, input FacetOptions) ([]point.FacetOutput, error)

	// EnsureCollection tạo collection nếu chưa tồn tại.
	EnsureCollection(ctx context.Context, name string, vectorSize uint64) error
}
```

---

### 3. `internal/point/repository/qdrant/collection.go` ← FILE MỚI

Cần xem `pkg/qdrant` interface trước để biết method nào available. Giả sử `IQdrant` có `CollectionExists` và `CreateCollection`:

```go
package qdrant

import (
	"context"

	pb "github.com/qdrant/go-client/qdrant"
)

// EnsureCollection creates a collection if it doesn't already exist.
// vectorSize phải match Voyage AI embedding dimension (1024).
// distance mặc định Cosine — phù hợp với semantic search.
func (r *implRepository) EnsureCollection(ctx context.Context, name string, vectorSize uint64) error {
	exists, err := r.client.CollectionExists(ctx, name)
	if err != nil {
		r.l.Errorf(ctx, "point.repository.qdrant.EnsureCollection: failed to check collection %s: %v", name, err)
		return err
	}
	if exists {
		return nil
	}

	r.l.Infof(ctx, "point.repository.qdrant.EnsureCollection: creating collection %s (vectorSize=%d)", name, vectorSize)
	if err := r.client.CreateCollection(ctx, name, vectorSize, pb.Distance_Cosine); err != nil {
		r.l.Errorf(ctx, "point.repository.qdrant.EnsureCollection: failed to create collection %s: %v", name, err)
		return err
	}

	r.l.Infof(ctx, "point.repository.qdrant.EnsureCollection: collection %s created successfully", name)
	return nil
}
```

**Lưu ý**: Xem `pkg/qdrant/interface.go` để verify method signature của `CollectionExists` và `CreateCollection`. Nếu tên method khác, điều chỉnh cho đúng.

---

### 4. `internal/point/usecase/collection.go` ← FILE MỚI

```go
package usecase

import "context"

const defaultVectorSize uint64 = 1024 // Voyage AI embedding dimension

// EnsureCollection delegates to repository.
func (uc *implUseCase) EnsureCollection(ctx context.Context, name string, vectorSize uint64) error {
	return uc.repo.EnsureCollection(ctx, name, vectorSize)
}
```

---

### 5. Gọi `EnsureCollection` trước khi Upsert

Trong `internal/indexing/usecase/index_insight.go` và `index_digest.go`, thêm `EnsureCollection` call trước bước Upsert:

**`index_insight.go` — Step 4.5 (thêm vào trước Step 5):**

```go
// Step 4.5: Ensure macro_insights collection exists
if err := uc.pointUC.EnsureCollection(ctx, kafkaDelivery.CollectionMacroInsights, 1024); err != nil {
    uc.l.Errorf(ctx, "indexing.usecase.IndexInsight: failed to ensure collection: %v", err)
    return indexing.IndexInsightOutput{}, err
}
```

**`index_digest.go` — Step 4.5 (thêm vào trước Step 5):**

```go
// Step 4.5: Ensure macro_insights collection exists
if err := uc.pointUC.EnsureCollection(ctx, kafkaDelivery.CollectionMacroInsights, 1024); err != nil {
    uc.l.Errorf(ctx, "indexing.usecase.IndexDigest: failed to ensure collection: %v", err)
    return indexing.IndexDigestOutput{}, err
}
```

**`index_batch.go` — Thêm vào đầu `IndexBatch()` sau validation:**

```go
// Ensure per-project collection exists
collectionName := fmt.Sprintf("proj_%s", input.ProjectID)
if err := uc.pointUC.EnsureCollection(ctx, collectionName, 1024); err != nil {
    uc.l.Errorf(ctx, "indexing.usecase.IndexBatch: failed to ensure collection %s: %v", collectionName, err)
    return indexing.IndexBatchOutput{}, err
}
```

---

### 6. Verify Qdrant Metadata Indexes

Sau khi collections được tạo, cần tạo metadata indexes để filter và facet hoạt động hiệu quả. Đây là **one-time setup** — có thể làm qua Qdrant dashboard hoặc startup script.

**Collection `macro_insights` indexes:**

```
project_id      → keyword
campaign_id     → keyword
rag_document_type → keyword   ← quan trọng nhất cho routing
insight_type    → keyword
confidence      → float
analysis_window_start → datetime
analysis_window_end   → datetime
```

**Collection `proj_{project_id}` indexes:**

```
project_id      → keyword
campaign_id     → keyword
sentiment_label → keyword
priority        → keyword
uap_type        → keyword
platform        → keyword
published_at    → datetime
impact_score    → float
```

Nếu muốn auto-create indexes qua code, thêm vào `EnsureCollection` sau `CreateCollection`. Xem `pkg/qdrant` để biết API.

---

## End-to-End Smoke Test

Sau khi M1–M6 hoàn tất, chạy sequence sau:

```bash
# 1. Produce Layer 3 (new format)
# Construct BatchCompletedMessage với 2-3 documents từ:
# documents/refactor-input/uap-batches/
# Produce vào: analytics.batch.completed

# 2. Produce Layer 2 (7 insight cards)
# Dùng data từ: documents/refactor-input/outputSRM/insights/insights.jsonl
# Mỗi line → 1 message vào: analytics.insights.published

# 3. Produce Layer 1 (1 digest)
# Dùng data từ: documents/refactor-input/outputSRM/reports/bi_reports.json
# Construct ReportDigestMessage và produce vào: analytics.report.digest
```

### Expected results:

| Check | Expected |
|-------|----------|
| `proj_{project_id}` collection exists | ✅ |
| `macro_insights` collection exists | ✅ |
| Points in `proj_{project_id}` | = số documents với rag=true |
| Points in `macro_insights` | 8 (7 insight_card + 1 report_digest) |
| `rag_document_type=insight_card` | 7 points |
| `rag_document_type=report_digest` | 1 point |
| Point ID format Layer 3 | `{uap_id}` |
| Point ID format Layer 2 | `insight:{run_id}:{type}:{hash12}` |
| Point ID format Layer 1 | `digest:{run_id}` |

---

## Final Verification Checklist

```bash
# Build
go build ./...
go vet ./...

# No hardcoded collection names
grep -rn "smap_analytics" internal/ --include="*.go"
# Expected: chỉ thấy ở search/aggregate.go (backward compat comments)

# No legacy const
grep -rn "const collectionName" internal/ --include="*.go"
# Expected: 0 results

# All 3 consumers start
# Observe startup logs:
# → "Started consuming topic: analytics.batch.completed (group: knowledge-indexing-batch)"
# → "Started consuming topic: analytics.insights.published (group: knowledge-indexing-insights)"
# → "Started consuming topic: analytics.report.digest (group: knowledge-indexing-digest)"

# Graceful shutdown
# Send SIGTERM → observe:
# → "All consumers stopped"
# → No error logs during shutdown
```

---

## Notes cho reviewer

- `EnsureCollection` được gọi mỗi lần message arrive — check `CollectionExists` rất nhanh (gRPC call ~1ms) nên không ảnh hưởng performance đáng kể
- Nếu muốn optimize: cache collection existence state trong memory sau lần đầu check thành công
- Voyage AI embedding dimension là **1024** — hardcode constant `defaultVectorSize = 1024` phải đúng
- Qdrant `Distance_Cosine` là chuẩn cho semantic search — không dùng `Dot` hay `Euclidean`
- Verify `pkg/qdrant/interface.go` có `CollectionExists(ctx, name) (bool, error)` và `CreateCollection(ctx, name, vectorSize, distance)` trước khi implement
