# Repository Layer Convention (`convention_repository.md`)

> **Role**: The Repository Layer handles **Data Access**.
> **Motto**: "One interface, multiple drivers, strict splitting, standard naming."

## 1. The "Split Pattern" (MANDATORY)

Repositories often become "God Objects" with 1000+ lines. To prevent this, strict file splitting is **ENFORCED**.

### The Triad

For every entity (e.g., `IndexedDocument`), you MUST have 3 files in `repository/<driver>/`:

1.  **`<entity>.go` (The Coordinator)**:
    - **Responsibility**: Orchestrates the flow.
    - **Code**: calls `buildQuery` -> calls Driver -> calls `toDomain`.
2.  **`<entity>_query.go` (The Builder)**:
    - **Responsibility**: Pure logic. Constructs filters (`bson.M` or `qm.QueryMod`).
    - **Code**: `if id != "" { filter["_id"] = id }`
3.  **`<entity>_mapper.go` (The Mapper)** (optional if using `internal/model` mappers):
    - **Responsibility**: Converts Data Models <-> Domain Models.
    - **Code**: `return models.IndexedDocument{ ID: dbDoc.ID }`

---

## 2. Standard Method Names (MANDATORY)

Repository methods MUST follow these **exact names**. Do NOT invent custom names.

### 2.1 CRUD Operations

| Method           | Purpose                 | Signature Example                                                           |
| ---------------- | ----------------------- | --------------------------------------------------------------------------- |
| **`Create`**     | Insert single record    | `Create(ctx, opt CreateOptions) (model.Entity, error)`                      |
| **`CreateMany`** | Bulk insert (rare)      | `CreateMany(ctx, opts []CreateOptions) ([]model.Entity, error)`             |
| **`Upsert`**     | Insert or update        | `Upsert(ctx, opt UpsertOptions) (model.Entity, error)`                      |
| **`Detail`**     | Get by ID only          | `Detail(ctx, id string) (model.Entity, error)`                              |
| **`GetOne`**     | Get by filters (unique) | `GetOne(ctx, opt GetOneOptions) (model.Entity, error)`                      |
| **`Get`**        | List with pagination    | `Get(ctx, opt GetOptions) ([]model.Entity, paginator.Paginator, error)`     |
| **`List`**       | List without pagination | `List(ctx, opt ListOptions) ([]model.Entity, error)`                        |
| **`Update`**     | Update by ID            | `Update(ctx, opt UpdateOptions) (model.Entity, error)`                      |
| **`Delete`**     | Delete single by ID     | `Delete(ctx, id string) error`                                              |
| **`Deletes`**    | Delete multiple         | `Deletes(ctx, ids []string) error`                                          |

**Important Notes:**

- ✅ **Scope in Context**: `model.Scope` is extracted from `context.Context` using `scope.GetScopeFromContext()`, NOT passed as parameter
- ❌ **NO `Exists` methods**: Use `GetOne` and check if result is empty
- ✅ **Return entities**: `Create`, `Update`, `Upsert` return the modified entity (value type)
- ✅ **`Get` returns `paginator.Paginator`**: NOT `int` for total count

### 2.2 Specialized Queries

For specialized queries, use descriptive names with context:

| Method             | Purpose               | Example                                              |
| ------------------ | --------------------- | ---------------------------------------------------- |
| **`Count`**        | Count records         | `CountByProject(ctx, projectID string) (int, error)` |
| **`UpdateStatus`** | Update specific field | `UpdateStatus(ctx, id string, status string) error`  |

**Avoid creating:**

- ❌ `ExistsByX` methods → Use `GetOne` instead
- ❌ `FindByX` methods → Use `GetOne` or `List` instead
- ❌ Custom query methods → Use `GetOne`, `List`, or `Get` with Options

---

## 3. Options Pattern (MANDATORY)

### 3.1 Flow: UseCase ↔ Repository

```
UseCase (domain models)
    ↓ Convert to Options
Repository Interface (Options)
    ↓ Use Options to build query
Driver (SQLBoiler/Mongo)
    ↓ Convert to domain models
UseCase (domain models)
```

### 3.2 Naming Convention

Options MUST be defined in `repository/option.go` with suffix `Options`:

```go
// repository/option.go

type CreateOptions struct {
    AnalyticsID string
    ProjectID   string
    SourceID    string
    // ... all fields needed for creation
}

type UpsertOptions struct {
    AnalyticsID string
    ProjectID   string
    SourceID    string
    // ... all fields needed for upsert
}

type GetOneOptions struct {
    AnalyticsID string  // optional filter
    ContentHash string  // optional filter
    // If both provided, they will be combined with AND
}

type GetOptions struct {
    // Filters
    Status      string
    ProjectID   string
    ErrorTypes  []string

    // Pagination (REQUIRED for Get)
    Limit  int
    Offset int

    // Sorting
    OrderBy string // e.g., "created_at DESC"
}

type ListOptions struct {
    // Filters only, NO pagination
    Status     string
    ProjectID  string
    MaxRetry   int
    ErrorTypes []string

    // Optional limit for safety
    Limit int // default: no limit, but can cap at 1000
}
```

### 3.3 Rules

- ✅ **DO**: Use `Options` for ALL operations (Create, Upsert, GetOne, Get, List)
- ✅ **DO**: Return `model.Entity` from repository (NOT SQLBoiler types)
- ✅ **DO**: Apply all filters provided in Options (AND condition)
- ❌ **DON'T**: Pass `model.Entity` as parameter to repository
- ❌ **DON'T**: Pass individual fields as separate parameters
- ❌ **DON'T**: Return SQLBoiler/Mongo types from repository interface
- ❌ **DON'T**: Add business logic in repository (validation, if/else for filter selection)

**Important**:

- **UseCase → Repository**: Always use `Options` structs
- **Repository → UseCase**: Always return `model.Entity` types
- Repository is a "dumb" data access layer. If `Options` has multiple filters, apply ALL of them with AND. Business logic (e.g., "only use one filter") belongs in UseCase layer.

---

## 4. PostgreSQL Implementation (SQLBoiler)

We use **SQLBoiler**. It generates type-safe Go structs from the DB schema.

### 4.1 Directory Structure

```text
repository/postgre/
├── new.go                    # Factory
├── indexed_document.go       # Coordinator: Create, GetOne, List, etc.
├── indexed_document_query.go # Builder: buildGetOneQuery, buildListQuery
├── dlq.go                    # Coordinator for DLQ entity
├── dlq_query.go              # Builder for DLQ
```

### 4.2 Method Implementation Pattern

#### Example: `Create`

```go
// indexed_document.go
func (r *implRepository) Create(ctx context.Context, opt CreateOptions) error {
    // Convert Options to SQLBoiler model
    dbDoc := &sqlboiler.IndexedDocument{
        AnalyticsID:    opt.AnalyticsID,
        ProjectID:      opt.ProjectID,
        SourceID:       opt.SourceID,
        QdrantPointID:  opt.QdrantPointID,
        CollectionName: opt.CollectionName,
        ContentHash:    opt.ContentHash,
        Status:         opt.Status,
        // ... map other fields
    }

    // Insert using SQLBoiler
    return dbDoc.Insert(ctx, r.db, boil.Infer())
}
```

#### Example: `Detail` (Get by ID only)

```go
// indexed_document.go
func (r *implRepository) Detail(ctx context.Context, id string) (*model.IndexedDocument, error) {
    // SQLBoiler's FindX method queries by primary key
    dbDoc, err := sqlboiler.FindIndexedDocument(ctx, r.db, id)
    if err == sql.ErrNoRows {
        return nil, nil // or return ErrNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("Detail: %w", err)
    }

    return model.NewIndexedDocumentFromDB(dbDoc), nil
}
```

#### Example: `GetOne` (Get by filters)

```go
// indexed_document.go
func (r *implRepository) GetOne(ctx context.Context, opt GetOneOptions) (*model.IndexedDocument, error) {
    // Build query mods
    mods := r.buildGetOneQuery(opt)

    // Execute
    dbDoc, err := sqlboiler.IndexedDocuments(mods...).One(ctx, r.db)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("GetOne: %w", err)
    }

    return model.NewIndexedDocumentFromDB(dbDoc), nil
}

// indexed_document_query.go
func (r *implRepository) buildGetOneQuery(opt GetOneOptions) []qm.QueryMod {
    mods := []qm.QueryMod{}

    // Apply ALL provided filters (AND condition)
    // Business logic to choose which filter belongs in UseCase
    if opt.AnalyticsID != "" {
        mods = append(mods, qm.Where("analytics_id = ?", opt.AnalyticsID))
    }
    if opt.ContentHash != "" {
        mods = append(mods, qm.Where("content_hash = ?", opt.ContentHash))
    }

    return mods
}
```

#### Example: `Get` (with pagination)

```go
// indexed_document.go
func (r *implRepository) Get(ctx context.Context, opt GetOptions) ([]model.IndexedDocument, int, error) {
    // 1. Count total
    countMods := r.buildGetCountQuery(opt)
    total, err := sqlboiler.IndexedDocuments(countMods...).Count(ctx, r.db)
    if err != nil {
        return nil, 0, fmt.Errorf("Get count: %w", err)
    }

    // 2. Get data
    mods := r.buildGetQuery(opt)
    dbDocs, err := sqlboiler.IndexedDocuments(mods...).All(ctx, r.db)
    if err != nil {
        return nil, 0, fmt.Errorf("Get: %w", err)
    }

    return r.toDomainList(dbDocs), int(total), nil
}

// indexed_document_query.go
func (r *implRepository) buildGetQuery(opt GetOptions) []qm.QueryMod {
    mods := []qm.QueryMod{}

    // Filters
    if opt.Status != "" {
        mods = append(mods, qm.Where("status = ?", opt.Status))
    }
    if opt.ProjectID != "" {
        mods = append(mods, qm.Where("project_id = ?", opt.ProjectID))
    }

    // Sorting
    if opt.OrderBy != "" {
        mods = append(mods, qm.OrderBy(opt.OrderBy))
    } else {
        mods = append(mods, qm.OrderBy("created_at DESC"))
    }

    // Pagination (REQUIRED)
    if opt.Limit > 0 {
        mods = append(mods, qm.Limit(opt.Limit))
    }
    if opt.Offset > 0 {
        mods = append(mods, qm.Offset(opt.Offset))
    }

    return mods
}
```

#### Example: `List` (no pagination)

```go
// indexed_document.go
func (r *implRepository) List(ctx context.Context, opt ListOptions) ([]model.IndexedDocument, error) {
    mods := r.buildListQuery(opt)

    dbDocs, err := sqlboiler.IndexedDocuments(mods...).All(ctx, r.db)
    if err != nil {
        return nil, fmt.Errorf("List: %w", err)
    }

    return r.toDomainList(dbDocs), nil
}

// indexed_document_query.go
func (r *implRepository) buildListQuery(opt ListOptions) []qm.QueryMod {
    mods := []qm.QueryMod{}

    // Filters
    if opt.Status != "" {
        mods = append(mods, qm.Where("status = ?", opt.Status))
    }
    if opt.MaxRetry > 0 {
        mods = append(mods, qm.Where("retry_count < ?", opt.MaxRetry))
    }

    // Safety limit (optional)
    if opt.Limit > 0 {
        mods = append(mods, qm.Limit(opt.Limit))
    }

    return mods
}
```

### 4.3 Critical Rules

- **Always use `qm`**: Never write raw SQL strings unless absolutely necessary (bulk insert, aggregations).
- **Context**: All DB calls must accept `context.Context`.
- **No Logic**: Do not put business logic (e.g., "defaults") in the repo.
- **Error Wrapping**: Use `fmt.Errorf("MethodName: %w", err)` for context.

---

## 5. Existence Checks: Use GetOne, NOT Exists

### ❌ WRONG: Creating Exists methods

```go
// DON'T do this
func (r *implRepository) ExistsByAnalyticsID(ctx context.Context, analyticsID string) (bool, error) {
    return sqlboiler.IndexedDocuments(
        qm.Where("analytics_id = ?", analyticsID),
    ).Exists(ctx, r.db)
}
```

### ✅ CORRECT: Use GetOne

```go
// In UseCase
doc, err := repo.GetOne(ctx, repository.GetOneOptions{
    AnalyticsID: analyticsID,
})
if err != nil {
    return err
}

if doc != nil {
    // Record exists
    return ErrDuplicate
}

// Record does not exist, proceed...
```

### Why?

1. **Simplicity**: One method (`GetOne`) instead of multiple `Exists` methods
2. **Flexibility**: If you need the data later, you already have it
3. **Performance**: Modern databases optimize `SELECT *` with `LIMIT 1` very well
4. **Consistency**: All queries go through the same pattern

### Performance Comparison

| Approach   | Query                        | Performance | Use Case                |
| ---------- | ---------------------------- | ----------- | ----------------------- |
| `Exists()` | `SELECT EXISTS(...)`         | ⚡ 1-2ms    | Only check existence    |
| `GetOne()` | `SELECT * WHERE ... LIMIT 1` | ⚡ 1-3ms    | Check + might need data |

**Difference: Negligible (< 1ms)** for indexed columns. Use `GetOne` for simplicity.

---

## 6. Exists vs GetOne vs Detail - Performance

### Comparison

| Method       | Query                   | Returns        | Use Case               | Performance            |
| ------------ | ----------------------- | -------------- | ---------------------- | ---------------------- |
| **`Exists`** | `SELECT EXISTS(...)`    | `bool`         | Check if record exists | ⚡ Fastest (1 bit)     |
| **`Detail`** | `SELECT * WHERE id = ?` | `model.Entity` | Get by ID              | ⚡⚡ Fast (indexed)    |
| **`GetOne`** | `SELECT * WHERE ...`    | `model.Entity` | Get by any filter      | ⚡⚡ Fast (if indexed) |

### Example Implementation

```go
// Exists - Use SQLBoiler's Exists() method
func (r *implRepository) ExistsByAnalyticsID(ctx context.Context, analyticsID string) (bool, error) {
    return sqlboiler.IndexedDocuments(
        qm.Where("analytics_id = ?", analyticsID),
    ).Exists(ctx, r.db)
}

// Detail - Use SQLBoiler's FindX() method (primary key lookup)
func (r *implRepository) Detail(ctx context.Context, id string) (*model.IndexedDocument, error) {
    dbDoc, err := sqlboiler.FindIndexedDocument(ctx, r.db, id)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("Detail: %w", err)
    }
    return model.NewIndexedDocumentFromDB(dbDoc), nil
}

// GetOne - Use SQLBoiler's One() method (any filter)
func (r *implRepository) GetOne(ctx context.Context, opt GetOneOptions) (*model.IndexedDocument, error) {
    mods := r.buildGetOneQuery(opt)
    dbDoc, err := sqlboiler.IndexedDocuments(mods...).One(ctx, r.db)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("GetOne: %w", err)
    }
    return model.NewIndexedDocumentFromDB(dbDoc), nil
}
```

### When to Use Which?

- **`Exists`**: Khi chỉ cần biết record có tồn tại không (duplicate check)
- **`Detail`**: Khi cần lấy record **bằng ID** (primary key)
- **`GetOne`**: Khi cần lấy record **bằng filter khác** (analytics_id, content_hash)

---

## 6. Interface Composition

Don't create a massive `Repository` interface. Split it by entity.

```go
// repository/interface.go

// The Main Interface (injected into UseCase)
type Repository interface {
    IndexedDocumentRepository
    DLQRepository
}

// Sub-Interface: IndexedDocument
type IndexedDocumentRepository interface {
    Create(ctx context.Context, opt CreateOptions) error
    Detail(ctx context.Context, id string) (*model.IndexedDocument, error)
    GetOne(ctx context.Context, opt GetOneOptions) (*model.IndexedDocument, error)
    Get(ctx context.Context, opt GetOptions) ([]model.IndexedDocument, int, error)
    List(ctx context.Context, opt ListOptions) ([]model.IndexedDocument, error)
    Upsert(ctx context.Context, opt UpsertOptions) error
    UpdateStatus(ctx context.Context, id string, status string, opt StatusMetrics) error
    CountByProject(ctx context.Context, projectID string) (ProjectStats, error)
}

// Sub-Interface: DLQ
type DLQRepository interface {
    CreateDLQ(ctx context.Context, opt CreateDLQOptions) error
    GetOneDLQ(ctx context.Context, opt GetOneDLQOptions) (*model.IndexingDLQ, error)
    ListDLQ(ctx context.Context, opt ListDLQOptions) ([]model.IndexingDLQ, error)
}
```

---

## 8. Common Patterns

### 8.1 Mapper Helper

```go
// indexed_document.go or indexed_document_mapper.go

func (r *implRepository) toDomainList(dbDocs []*sqlboiler.IndexedDocument) []model.IndexedDocument {
    result := make([]model.IndexedDocument, 0, len(dbDocs))
    for _, db := range dbDocs {
        if doc := model.NewIndexedDocumentFromDB(db); doc != nil {
            result = append(result, *doc)
        }
    }
    return result
}
```

### 8.2 Transaction Support (Optional)

```go
type Repository interface {
    WithTx(tx *sql.Tx) Repository
}

func (r *implRepository) WithTx(tx *sql.Tx) Repository {
    return &implRepository{
        db: tx, // Use transaction instead of DB
    }
}
```

---

## 9. Intern Checklist (Read before PR)

- [ ] **Naming Check**: Did I use standard method names (`Create`, `GetOne`, `List`, etc.)?
- [ ] **No Exists**: Did I avoid creating `Exists` methods? Use `GetOne` instead!
- [ ] **Options Check**: Did I define all Options in `option.go`?
- [ ] **Split Check**: Did I split my code into `entity.go` and `entity_query.go`?
- [ ] **Context Check**: Am I passing `ctx` to the driver?
- [ ] **Nil Check**: Do I return `nil` (not error) when record not found in `GetOne`/`Detail`?
- [ ] **Mapping Check**: Did I handle `null` fields correctly? (e.g., `null.String` -> `*string`).
- [ ] **Leak Check**: Am I returning `bson.M` or `sqlboiler.IndexedDocument` out of the interface? **FORBIDDEN**. Return `internal/model` types.
- [ ] **Error Wrapping**: Did I wrap errors with method name for debugging?
