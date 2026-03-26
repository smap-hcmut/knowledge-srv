# M1: Dynamic Collection Name — Point Domain

**Prerequisite**: None — đây là foundation, implement TRƯỚC tất cả milestones khác.

**Goal**: Xóa `const collectionName = "smap_analytics"` hardcoded trong point domain. Mọi operation Qdrant phải nhận collection name qua parameter.

**Risk**: Thấp — chỉ thêm field, callers vẫn compile với zero value `""`. Verify bằng `go build ./...` sau mỗi file.

---

## Files cần thay đổi (theo thứ tự)

### 1. `internal/point/types.go`

Thêm `CollectionName string` vào TẤT CẢ input structs.

```go
package point

import (
	"knowledge-srv/internal/model"

	"github.com/qdrant/go-client/qdrant"
)

type Filter = qdrant.Filter

type SearchInput struct {
	CollectionName string // NEW: required — e.g. "proj_cleanser_01" or "macro_insights"
	Vector         []float32
	Filter         *Filter
	Limit          uint64
	WithPayload    bool
	ScoreThreshold float32
}

type SearchOutput struct {
	ID      string
	Score   float32
	Payload map[string]interface{}
}

type UpsertInput struct {
	CollectionName string       // NEW: required
	Points         []model.Point
}

type CountInput struct {
	CollectionName string // NEW: required
	Filter         *Filter
}

type DeleteInput struct {
	CollectionName string // NEW: required
	Filter         *Filter
	Points         []string
}

type ScrollInput struct {
	CollectionName string // NEW: required
	Filter         *Filter
	Limit          uint64
	WithPayload    bool
	Offset         *string
}

type FacetInput struct {
	CollectionName string // NEW: required
	Key            string
	Filter         *Filter
	Limit          uint64
}

type FacetOutput struct {
	Value string
	Count uint64
}
```

---

### 2. `internal/point/repository/options.go`

Thêm `CollectionName string` vào TẤT CẢ options structs (mirror của types.go).

```go
package repository

import (
	"knowledge-srv/internal/model"

	"github.com/qdrant/go-client/qdrant"
)

type SearchOptions struct {
	CollectionName string // NEW
	Vector         []float32
	Filter         *qdrant.Filter
	Limit          uint64
	WithPayload    bool
	ScoreThreshold float32
}

type UpsertOptions struct {
	CollectionName string // NEW
	Points         []model.Point
}

type CountOptions struct {
	CollectionName string // NEW
	Filter         *qdrant.Filter
}

type DeleteOptions struct {
	CollectionName string // NEW
	Filter         *qdrant.Filter
	Points         []string
}

type ScrollOptions struct {
	CollectionName string // NEW
	Filter         *qdrant.Filter
	Limit          uint64
	WithPayload    bool
	Offset         *string
}

type FacetOptions struct {
	CollectionName string // NEW
	Key            string
	Filter         *qdrant.Filter
	Limit          uint64
}
```

---

### 3. `internal/point/repository/qdrant/point.go`

**XÓA** `const collectionName = "smap_analytics"`. Thay bằng `opt.CollectionName` ở mỗi method.

```go
package qdrant

import (
	"context"
	"fmt"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
	pkgQdrant "knowledge-srv/pkg/qdrant"
)

// collectionName constant đã bị XÓA — collection name phải truyền qua options

func (r *implRepository) Search(ctx context.Context, opt repository.SearchOptions) ([]point.SearchOutput, error) {
	pkgResults, err := r.client.SearchWithFilter(ctx, opt.CollectionName, opt.Vector, opt.Limit, opt.Filter)
	if err != nil {
		r.l.Errorf(ctx, "point.repository.qdrant.Search: Failed to search points: %v", err)
		return nil, err
	}

	results := make([]point.SearchOutput, len(pkgResults))
	for i, pr := range pkgResults {
		results[i] = point.SearchOutput{
			ID:      pr.ID,
			Score:   pr.Score,
			Payload: pr.Payload,
		}
	}
	return results, nil
}

func (r *implRepository) Upsert(ctx context.Context, opt repository.UpsertOptions) error {
	pkgPoints := make([]pkgQdrant.Point, len(opt.Points))
	for i, p := range opt.Points {
		pkgPoints[i] = pkgQdrant.Point{
			ID:      p.ID,
			Vector:  p.Vector,
			Payload: p.Payload,
		}
	}
	if err := r.client.UpsertPoints(ctx, opt.CollectionName, pkgPoints); err != nil {
		r.l.Errorf(ctx, "point.repository.qdrant.Upsert: Failed to upsert points: %v", err)
		return err
	}
	return nil
}

func (r *implRepository) Count(ctx context.Context, opt repository.CountOptions) (uint64, error) {
	return r.client.CountPoints(ctx, opt.CollectionName)
}

func (r *implRepository) Delete(ctx context.Context, opt repository.DeleteOptions) error {
	return fmt.Errorf("not implemented")
}

func (r *implRepository) Scroll(ctx context.Context, opt repository.ScrollOptions) ([]model.Point, error) {
	return nil, fmt.Errorf("not implemented")
}
```

---

### 4. `internal/point/repository/qdrant/facet.go`

Thay `collectionName` constant bằng `input.CollectionName`.

```go
package qdrant

import (
	"context"
	"fmt"

	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
)

func (r *implRepository) Facet(ctx context.Context, input repository.FacetOptions) ([]point.FacetOutput, error) {
	pkgResults, err := r.client.Facet(ctx, input.CollectionName, input.Key, input.Limit, input.Filter)
	if err != nil {
		r.l.Errorf(ctx, "point.repository.qdrant.Facet: Failed to facet points: %v", err)
		return nil, err
	}

	results := make([]point.FacetOutput, len(pkgResults))
	for i, pr := range pkgResults {
		var valStr string
		switch v := pr.Value.(type) {
		case string:
			valStr = v
		case int64:
			valStr = fmt.Sprintf("%d", v)
		case int:
			valStr = fmt.Sprintf("%d", v)
		case float64:
			valStr = fmt.Sprintf("%.2f", v)
		default:
			if v != nil {
				valStr = fmt.Sprintf("%v", v)
			}
		}

		results[i] = point.FacetOutput{
			Value: valStr,
			Count: pr.Count,
		}
	}

	return results, nil
}
```

---

### 5. `internal/point/usecase/upsert.go`

Pass `CollectionName` từ input → options.

```go
package usecase

import (
	"context"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
)

func (uc *implUseCase) Upsert(ctx context.Context, input point.UpsertInput) error {
	return uc.repo.Upsert(ctx, repository.UpsertOptions{
		CollectionName: input.CollectionName, // NEW
		Points:         input.Points,
	})
}
```

---

### 6. `internal/point/usecase/search.go`

```go
package usecase

import (
	"context"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
)

func (uc *implUseCase) Search(ctx context.Context, input point.SearchInput) ([]point.SearchOutput, error) {
	return uc.repo.Search(ctx, repository.SearchOptions{
		CollectionName: input.CollectionName, // NEW
		Vector:         input.Vector,
		Filter:         input.Filter,
		Limit:          input.Limit,
		WithPayload:    input.WithPayload,
		ScoreThreshold: input.ScoreThreshold,
	})
}
```

---

### 7. `internal/point/usecase/count.go`

```go
package usecase

import (
	"context"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
)

func (uc *implUseCase) Count(ctx context.Context, input point.CountInput) (uint64, error) {
	return uc.repo.Count(ctx, repository.CountOptions{
		CollectionName: input.CollectionName, // NEW
		Filter:         input.Filter,
	})
}
```

---

### 8. `internal/point/usecase/delete.go`

```go
package usecase

import (
	"context"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
)

func (uc *implUseCase) Delete(ctx context.Context, input point.DeleteInput) error {
	return uc.repo.Delete(ctx, repository.DeleteOptions{
		CollectionName: input.CollectionName, // NEW
		Filter:         input.Filter,
		Points:         input.Points,
	})
}
```

---

### 9. `internal/point/usecase/scroll.go`

```go
package usecase

import (
	"context"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
)

func (uc *implUseCase) Scroll(ctx context.Context, input point.ScrollInput) ([]model.Point, error) {
	return uc.repo.Scroll(ctx, repository.ScrollOptions{
		CollectionName: input.CollectionName, // NEW
		Filter:         input.Filter,
		Limit:          input.Limit,
		WithPayload:    input.WithPayload,
		Offset:         input.Offset,
	})
}
```

---

### 10. `internal/point/usecase/facet.go`

```go
package usecase

import (
	"context"

	"knowledge-srv/internal/point"
	"knowledge-srv/internal/point/repository"
)

func (uc *implUseCase) Facet(ctx context.Context, input point.FacetInput) ([]point.FacetOutput, error) {
	return uc.repo.Facet(ctx, repository.FacetOptions{
		CollectionName: input.CollectionName, // NEW
		Key:            input.Key,
		Filter:         input.Filter,
		Limit:          input.Limit,
	})
}
```

---

### 11. `internal/search/usecase/search.go` (caller — backward compat)

Tìm dòng 73 gọi `pointUC.Search()`. Thêm `CollectionName` với constant `"smap_analytics"` để giữ backward compat.

**Chỉ thay đổi phần gọi Search, không đổi gì khác:**

```go
// Step 5: Search Qdrant (Via Point Domain)
pointResults, err := uc.pointUC.Search(ctx, point.SearchInput{
    CollectionName: "smap_analytics", // backward compat — sẽ migrate sau khi Layer 3 dùng proj_{id}
    Vector:         vector,
    Filter:         filter,
    Limit:          uint64(limit),
    WithPayload:    true,
    ScoreThreshold: 0,
})
```

---

### 12. `internal/search/usecase/aggregate.go` (caller — backward compat)

Tìm 3 lần gọi `pointUC.Count()` và `pointUC.Facet()`. Thêm `CollectionName: "smap_analytics"` vào mỗi cái.

```go
// Task 1: Total Docs
count, err := uc.pointUC.Count(ctx, point.CountInput{
    CollectionName: "smap_analytics", // backward compat
    Filter:         baseFilter,
})

// Task 2: Sentiment Breakdown
res, err := uc.pointUC.Facet(ctx, point.FacetInput{
    CollectionName: "smap_analytics", // backward compat
    Key:            "overall_sentiment",
    Filter:         baseFilter,
    Limit:          10,
})

// Task 3: Platform Breakdown
res, err := uc.pointUC.Facet(ctx, point.FacetInput{
    CollectionName: "smap_analytics", // backward compat
    Key:            "platform",
    Filter:         baseFilter,
    Limit:          10,
})

// Task 4: Top Negative Aspects
res, err := uc.pointUC.Facet(ctx, point.FacetInput{
    CollectionName: "smap_analytics", // backward compat
    Key:            "aspects.aspect",
    Filter:         negFilter,
    Limit:          5,
})
```

---

### 13. `internal/indexing/usecase/index.go` (caller — dùng collection đúng)

Tìm dòng 222 gọi `pointUC.Upsert()`. Thêm `CollectionName` với collection name per-project:

```go
// Step 7: Upsert to Qdrant (Via Point Domain)
upsertStart := time.Now()
err = uc.pointUC.Upsert(ctx, point.UpsertInput{
    CollectionName: fmt.Sprintf("proj_%s", ip.ProjectID), // Layer 3: per-project collection
    Points: []model.Point{
        {
            ID:      pointID,
            Vector:  vector,
            Payload: payload,
        },
    },
})
```

**Lưu ý**: `ip.ProjectID` đã có trong `indexing.IndexInput` struct. Nếu chưa có `"fmt"` trong import, thêm vào.

---

## Verification Checklist (chạy sau khi implement xong M1)

```bash
# 1. Build phải pass
go build ./...

# 2. Không còn hardcoded collection name
grep -rn "smap_analytics" internal/ --include="*.go"
# Expected: chỉ thấy các comment backward compat trong search usecase

# 3. Không còn const collectionName
grep -rn "const collectionName" internal/ --include="*.go"
# Expected: 0 kết quả
```

---

## Notes cho reviewer (tôi — master agent)

- Kiểm tra `internal/point/repository/qdrant/point.go`: `const collectionName` phải được xóa hoàn toàn
- Kiểm tra tất cả 4 callers đã có `CollectionName` field
- `search` domain dùng `"smap_analytics"` là intentional backward compat — sẽ update ở M5 khi search cần target `proj_{id}`
- `indexing` domain dùng `fmt.Sprintf("proj_%s", ...)` là behavior mới đúng cho Layer 3
