# RAG Data Flow Chi Tiết - Từ Analyst đến RAG Response

**Mục đích:** Làm rõ flow data từ khi analyst upload file cho đến khi RAG có thể trả lời câu hỏi

---

## PHẦN 1: TỔNG QUAN FLOW DATA

### Câu hỏi cần trả lời:

1. ✅ **RAG gắn với Campaign như thế nào?**
2. ✅ **Flow data chi tiết từ analyst đến RAG?**
3. ✅ **Cách setup RAG đơn giản với 2 models?**

---

## PHẦN 2: CAMPAIGN-SCOPED RAG - GIẢI THÍCH CHI TIẾT

### 2.1 Mô hình Entity Hierarchy (3 tầng)

```
┌─────────────────────────────────────────────────────────────────┐
│                    ENTITY HIERARCHY                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  TẦNG 3: CAMPAIGN (Logical Analysis Unit)                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Campaign "So sánh Xe điện Q1/2026"                     │    │
│  │  campaign_id: "camp_001"                                │    │
│  │                                                         │    │
│  │  ├── Project "Monitor VF8"  (project_id: "proj_vf8")   │    │
│  │  │   brand: "VinFast"                                  │    │
│  │  │                                                      │    │
│  │  └── Project "Monitor BYD Seal" (project_id: "proj_byd")│   │
│  │      brand: "BYD"                                       │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↑                                     │
│  TẦNG 2: PROJECT (Entity Monitoring Unit)                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Project "Monitor VF8" (proj_vf8)                       │    │
│  │                                                         │    │
│  │  ├── Data Source 1: Excel "Feedback Q1.xlsx"           │    │
│  │  │   source_id: "src_001"                              │    │
│  │  │   → 500 UAP records                                 │    │
│  │  │                                                      │    │
│  │  ├── Data Source 2: TikTok Crawl "vinfast vf8"         │    │
│  │  │   source_id: "src_002"                              │    │
│  │  │   → 1000 UAP records                                │    │
│  │  │                                                      │    │
│  │  └── Data Source 3: Webhook từ CRM                     │    │
│  │      source_id: "src_003"                              │    │
│  │      → 300 UAP records (ongoing)                       │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↑                                     │
│  TẦNG 1: DATA SOURCE (Physical Data Unit)                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Data Source "Excel Feedback Q1.xlsx" (src_001)        │    │
│  │                                                         │    │
│  │  Raw File: feedback_q1.xlsx                             │    │
│  │  ├── Column A: "Ý kiến khách hàng"                     │    │
│  │  ├── Column B: "Ngày gửi"                              │    │
│  │  └── Column C: "Tên KH"                                │    │
│  │                                                         │    │
│  │  AI Schema Mapping:                                     │    │
│  │  ├── "Ý kiến khách hàng" → content                     │    │
│  │  ├── "Ngày gửi" → content_created_at                   │    │
│  │  └── "Tên KH" → metadata.author                        │    │
│  │                                                         │    │
│  │  Output: 500 UAP records                                │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Campaign Database Schema

```sql
-- Campaign table
CREATE TABLE business.campaigns (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Campaign-Project relationship (many-to-many)
CREATE TABLE business.campaign_projects (
    campaign_id UUID REFERENCES business.campaigns(id),
    project_id UUID REFERENCES business.projects(id),
    added_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (campaign_id, project_id)
);

-- Example data
INSERT INTO business.campaigns VALUES 
('camp_001', 'So sánh Xe điện Q1/2026', 'So sánh VinFast vs BYD', 'user_123', NOW());

INSERT INTO business.campaign_projects VALUES
('camp_001', 'proj_vf8', NOW()),
('camp_001', 'proj_byd', NOW());
```

### 2.3 RAG Query với Campaign Scope

```go
// Khi user hỏi trong Campaign "camp_001"
func (h *Handler) ChatWithCampaign(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
    // Step 1: Lấy danh sách projects trong campaign
    query := `
        SELECT project_id 
        FROM business.campaign_projects 
        WHERE campaign_id = $1
    `
    rows, _ := h.db.Query(ctx, query, req.CampaignID)
    
    projectIDs := []string{}
    for rows.Next() {
        var projectID string
        rows.Scan(&projectID)
        projectIDs = append(projectIDs, projectID)
    }
    // projectIDs = ["proj_vf8", "proj_byd"]
    
    // Step 2: Search trong Qdrant với filter project_id
    searchReq := &qdrant.SearchPoints{
        CollectionName: "smap_analytics",
        Vector:         queryVector,  // Embedding của câu hỏi
        Filter: &qdrant.Filter{
            Must: []*qdrant.Condition{
                {
                    Field: &qdrant.FieldCondition{
                        Key: "project_id",
                        Match: &qdrant.Match{
                            Keywords: &qdrant.RepeatedStrings{
                                Strings: projectIDs,  // CHỈ search trong proj_vf8 và proj_byd
                            },
                        },
                    },
                },
            },
        },
        Limit: 10,
    }
    
    results, _ := h.qdrantClient.Search(ctx, searchReq)
    
    // Step 3: Generate answer từ results
    answer := h.generateAnswer(ctx, req.Query, results)
    
    return &ChatResponse{Answer: answer}, nil
}
```

**Kết quả:** RAG CHỈ trả lời dựa trên data từ VF8 và BYD, KHÔNG lấy data từ projects khác.

---


## PHẦN 3: FLOW DATA CHI TIẾT - TỪ ANALYST ĐẾN RAG

### 3.1 Timeline Overview

```
T0: Analyst upload file
    ↓
T1: AI Schema Agent suggest mapping (5-10 giây)
    ↓
T2: Analyst confirm mapping
    ↓
T3: Transform raw data → UAP (10-30 giây)
    ↓
T4: Analytics Pipeline xử lý (Sentiment + Aspect) (1-5 phút)
    ↓
T5: Vector Indexing vào Qdrant (10-30 giây)
    ↓
T6: RAG sẵn sàng trả lời ✅
```

### 3.2 Chi tiết từng bước

#### BƯỚC 1: Analyst Upload File (T0)

```
┌─────────────────────────────────────────────────────────────────┐
│  WEB UI (React)                                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Campaign: "So sánh Xe điện Q1/2026"                    │    │
│  │  Project: "Monitor VF8"                                 │    │
│  │                                                         │    │
│  │  [Thêm Data Source]                                     │    │
│  │    ├── Loại: File Upload                               │    │
│  │    └── File: feedback_q1.xlsx (2MB, 500 rows)          │    │
│  │                                                         │    │
│  │  [Upload] ← Analyst click                              │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↓                                     │
│  HTTP POST /api/v1/sources/:id/upload                           │
│  Content-Type: multipart/form-data                              │
│  Body: file binary                                              │
└─────────────────────────────────────────────────────────────────┘
```

**Backend (Ingest Service):**

```go
// internal/api/upload.go
func (h *Handler) UploadFile(c *gin.Context) {
    // 1. Receive file
    file, _ := c.FormFile("file")
    
    // 2. Save to MinIO (temporary storage)
    minioPath := fmt.Sprintf("/temp/%s/%s", projectID, file.Filename)
    h.minioClient.PutObject(ctx, "smap-data", minioPath, file, file.Size)
    
    // 3. Parse file (Excel/CSV)
    rows := h.excelParser.Parse(file)
    // rows = [
    //   {"Ý kiến khách hàng": "Xe đẹp nhưng pin yếu", "Ngày gửi": "01/02/2026", "Tên KH": "Nguyễn A"},
    //   {"Ý kiến khách hàng": "Giá hơi cao", "Ngày gửi": "02/02/2026", "Tên KH": "Trần B"},
    //   ...
    // ]
    
    // 4. Extract header + 5 sample rows
    header := rows[0]  // ["Ý kiến khách hàng", "Ngày gửi", "Tên KH"]
    samples := rows[1:6]
    
    // 5. Call AI Schema Agent
    mapping := h.aiSchemaAgent.SuggestMapping(header, samples)
    
    // 6. Return mapping suggestion to UI
    c.JSON(200, gin.H{
        "source_id": sourceID,
        "mapping": mapping,
        "sample_data": samples,
    })
}
```

#### BƯỚC 2: AI Schema Agent Suggest Mapping (T1 - 5-10 giây)

```
┌─────────────────────────────────────────────────────────────────┐
│  AI SCHEMA AGENT (Python Sidecar hoặc Go + OpenAI)             │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Input:                                                 │    │
│  │  - Header: ["Ý kiến khách hàng", "Ngày gửi", "Tên KH"] │    │
│  │  - Samples: [                                           │    │
│  │      {"Ý kiến khách hàng": "Xe đẹp...", ...},          │    │
│  │      {"Ý kiến khách hàng": "Giá cao...", ...}          │    │
│  │    ]                                                    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↓                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  LLM Prompt (OpenAI GPT-4):                             │    │
│  │                                                         │    │
│  │  "Bạn là AI Schema Mapper. Dưới đây là header và       │    │
│  │   sample data từ file Excel. Hãy map các cột sang      │    │
│  │   UAP schema:                                           │    │
│  │                                                         │    │
│  │   UAP Schema:                                           │    │
│  │   - content (required): Nội dung chính để phân tích    │    │
│  │   - content_created_at: Thời gian tạo                  │    │
│  │   - metadata.author: Tác giả                           │    │
│  │   - metadata.*: Các field khác                         │    │
│  │                                                         │    │
│  │   Header: ['Ý kiến khách hàng', 'Ngày gửi', 'Tên KH']  │    │
│  │   Samples: [...]                                        │    │
│  │                                                         │    │
│  │   Trả về JSON mapping với confidence score."           │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↓                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  LLM Response:                                          │    │
│  │  {                                                      │    │
│  │    "mapping": {                                         │    │
│  │      "Ý kiến khách hàng": {                            │    │
│  │        "uap_field": "content",                         │    │
│  │        "confidence": 0.95                              │    │
│  │      },                                                 │    │
│  │      "Ngày gửi": {                                     │    │
│  │        "uap_field": "content_created_at",              │    │
│  │        "confidence": 0.90,                             │    │
│  │        "transform": "parse_date"                       │    │
│  │      },                                                 │    │
│  │      "Tên KH": {                                       │    │
│  │        "uap_field": "metadata.author",                 │    │
│  │        "confidence": 0.85                              │    │
│  │      }                                                  │    │
│  │    }                                                    │    │
│  │  }                                                      │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

**UI hiển thị:**

```
┌─────────────────────────────────────────────────────────────────┐
│  Xác nhận Schema Mapping                                        │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Cột gốc              │ UAP Field         │ Confidence  │    │
│  ├───────────────────────┼───────────────────┼─────────────┤    │
│  │ Ý kiến khách hàng     │ content           │ 95% ✅      │    │
│  │ Ngày gửi              │ content_created_at│ 90% ✅      │    │
│  │ Tên KH                │ metadata.author   │ 85% ✅      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  [Chỉnh sửa] [Xác nhận] ← Analyst click "Xác nhận"             │
└─────────────────────────────────────────────────────────────────┘
```

#### BƯỚC 3: Transform Raw Data → UAP (T3 - 10-30 giây)

```go
// Sau khi analyst confirm mapping
func (h *Handler) ConfirmMapping(c *gin.Context) {
    var req ConfirmMappingRequest
    c.BindJSON(&req)
    
    // 1. Save mapping rules to database
    h.db.Exec(`
        UPDATE ingest.data_sources 
        SET schema_mapping = $1, onboarding_status = 'CONFIRMED'
        WHERE id = $2
    `, req.Mapping, req.SourceID)
    
    // 2. Start transformation job (async)
    job := &TransformJob{
        SourceID: req.SourceID,
        Mapping:  req.Mapping,
    }
    h.jobQueue.Publish("ingest.transform.start", job)
    
    c.JSON(200, gin.H{"status": "processing"})
}

// Worker xử lý transformation
func (w *TransformWorker) Process(job *TransformJob) {
    // 1. Load raw data from MinIO
    rows := w.loadRawData(job.SourceID)
    
    // 2. Transform từng row → UAP
    uapRecords := []UAP{}
    for _, row := range rows {
        uap := UAP{
            ID:        uuid.New().String(),
            ProjectID: job.ProjectID,
            SourceID:  job.SourceID,
            Platform:  "internal_excel",
        }
        
        // Apply mapping rules
        for sourceCol, mapping := range job.Mapping {
            value := row[sourceCol]
            
            switch mapping.UAPField {
            case "content":
                uap.Content = value
            case "content_created_at":
                // Parse date: "01/02/2026" → "2026-02-01T00:00:00Z"
                uap.ContentCreatedAt = w.parseDate(value)
            case "metadata.author":
                uap.Metadata["author"] = value
            }
        }
        
        // Set ingested_at (system time)
        uap.IngestedAt = time.Now().UTC()
        
        uapRecords = append(uapRecords, uap)
    }
    
    // 3. Publish UAP records to Kafka
    for _, uap := range uapRecords {
        w.kafka.Publish("analytics.uap.received", uap)
    }
    
    // 4. Update source status
    w.db.Exec(`
        UPDATE ingest.data_sources 
        SET status = 'completed', record_count = $1
        WHERE id = $2
    `, len(uapRecords), job.SourceID)
}
```

**Kết quả:** 500 UAP records được publish vào Kafka topic `analytics.uap.received`

```json
// Example UAP record
{
  "id": "uap_001",
  "project_id": "proj_vf8",
  "source_id": "src_001",
  "content": "Xe đẹp nhưng pin yếu",
  "content_created_at": "2026-02-01T00:00:00Z",
  "ingested_at": "2026-02-15T10:30:00Z",
  "platform": "internal_excel",
  "metadata": {
    "author": "Nguyễn A"
  }
}
```

#### BƯỚC 4: Analytics Pipeline - Sentiment + Aspect (T4 - 1-5 phút)

```
┌─────────────────────────────────────────────────────────────────┐
│  ANALYTICS SERVICE (Go Consumer + AI Workers)                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Kafka Consumer                                         │    │
│  │  Topic: analytics.uap.received                          │    │
│  │                                                         │    │
│  │  Nhận UAP record → Process                              │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↓                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Orchestrator (Go)                                      │    │
│  │                                                         │    │
│  │  1. Call Sentiment Worker (Python FastAPI)             │    │
│  │     POST http://sentiment-worker:8000/analyze           │    │
│  │     Body: {"content": "Xe đẹp nhưng pin yếu"}          │    │
│  │     Response: {"sentiment": "NEGATIVE", "score": -0.6} │    │
│  │                                                         │    │
│  │  2. Call Aspect Worker (Python FastAPI) - PARALLEL     │    │
│  │     POST http://aspect-worker:8000/analyze              │    │
│  │     Body: {"content": "Xe đẹp nhưng pin yếu"}          │    │
│  │     Response: {                                         │    │
│  │       "aspects": [                                      │    │
│  │         {"aspect": "DESIGN", "sentiment": "POSITIVE",   │    │
│  │          "score": 0.7, "keywords": ["đẹp"]},           │    │
│  │         {"aspect": "PIN", "sentiment": "NEGATIVE",      │    │
│  │          "score": -0.8, "keywords": ["yếu"]}           │    │
│  │       ]                                                 │    │
│  │     }                                                    │    │
│  │                                                         │    │
│  │  3. Call Keyword Worker (Python FastAPI) - PARALLEL    │    │
│  │     POST http://keyword-worker:8000/extract             │    │
│  │     Response: {"keywords": ["xe", "đẹp", "pin", "yếu"]}│    │
│  │                                                         │    │
│  │  4. Aggregate results                                   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↓                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Save to PostgreSQL (analytics.post_analytics)          │    │
│  │  {                                                      │    │
│  │    "id": "analytics_001",                               │    │
│  │    "project_id": "proj_vf8",                            │    │
│  │    "source_id": "src_001",                              │    │
│  │    "content": "Xe đẹp nhưng pin yếu",                  │    │
│  │    "content_created_at": "2026-02-01T00:00:00Z",       │    │
│  │    "overall_sentiment": "NEGATIVE",                     │    │
│  │    "overall_sentiment_score": -0.6,                    │    │
│  │    "aspects": [                                         │    │
│  │      {"aspect": "DESIGN", "sentiment": "POSITIVE", ...},│    │
│  │      {"aspect": "PIN", "sentiment": "NEGATIVE", ...}   │    │
│  │    ],                                                   │    │
│  │    "keywords": ["xe", "đẹp", "pin", "yếu"]            │    │
│  │  }                                                      │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

**Quan trọng:** CHỈ SAU BƯỚC NÀY, data mới có đủ labels (sentiment + aspects) để đưa vào Qdrant!

#### BƯỚC 5: Vector Indexing vào Qdrant (T5 - 10-30 giây)

```
┌─────────────────────────────────────────────────────────────────┐
│  KNOWLEDGE SERVICE                                              │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Trigger: Analytics Service gọi sau khi save DB        │    │
│  │  POST /knowledge/index                                  │    │
│  │  Body: {                                                │    │
│  │    "project_id": "proj_vf8",                            │    │
│  │    "content": "Xe đẹp nhưng pin yếu",                  │    │
│  │    "sentiment": "NEGATIVE",                             │    │
│  │    "aspects": [...],                                    │    │
│  │    ...                                                  │    │
│  │  }                                                      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↓                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  1. Generate Embedding (OpenAI)                         │    │
│  │     POST https://api.openai.com/v1/embeddings           │    │
│  │     Body: {                                             │    │
│  │       "model": "text-embedding-3-small",                │    │
│  │       "input": "Xe đẹp nhưng pin yếu"                  │    │
│  │     }                                                    │    │
│  │     Response: {                                         │    │
│  │       "data": [{                                        │    │
│  │         "embedding": [0.123, -0.456, 0.789, ...]       │    │
│  │       }]                                                │    │
│  │     }                                                    │    │
│  │     → Vector có 1536 dimensions                         │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↓                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  2. Upsert to Qdrant                                    │    │
│  │     POST http://qdrant:6333/collections/smap_analytics/ │    │
│  │          points/upsert                                  │    │
│  │     Body: {                                             │    │
│  │       "points": [{                                      │    │
│  │         "id": "uap_001",                                │    │
│  │         "vector": [0.123, -0.456, ...],                │    │
│  │         "payload": {                                    │    │
│  │           "project_id": "proj_vf8",                     │    │
│  │           "content": "Xe đẹp nhưng pin yếu",           │    │
│  │           "sentiment": "NEGATIVE",                      │    │
│  │           "sentiment_score": -0.6,                     │    │
│  │           "aspects": [                                  │    │
│  │             {"aspect": "DESIGN", "sentiment": "POSITIVE"},│  │
│  │             {"aspect": "PIN", "sentiment": "NEGATIVE"}  │    │
│  │           ],                                            │    │
│  │           "content_created_at": 1738368000              │    │
│  │         }                                                │    │
│  │       }]                                                │    │
│  │     }                                                    │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

**Kết quả:** Vector được lưu trong Qdrant với đầy đủ metadata (project_id, sentiment, aspects)

#### BƯỚC 6: RAG Sẵn Sàng Trả Lời (T6) ✅

```
┌─────────────────────────────────────────────────────────────────┐
│  USER QUERY IN CAMPAIGN                                         │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Campaign: "So sánh Xe điện Q1/2026"                    │    │
│  │  Projects: ["proj_vf8", "proj_byd"]                     │    │
│  │                                                         │    │
│  │  User: "VinFast bị đánh giá tiêu cực về gì?"           │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↓                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  KNOWLEDGE SERVICE                                      │    │
│  │                                                         │    │
│  │  1. Embed query → vector                                │    │
│  │  2. Search Qdrant:                                      │    │
│  │     - Vector similarity                                 │    │
│  │     - Filter: project_id IN ["proj_vf8"]               │    │
│  │     - Filter: sentiment = "NEGATIVE"                    │    │
│  │     → Top 10 results                                    │    │
│  │                                                         │    │
│  │  3. Build context from results                          │    │
│  │  4. Call OpenAI GPT-4:                                  │    │
│  │     Prompt: "Dựa trên context sau, trả lời câu hỏi..." │    │
│  │     Context: [10 negative comments về VinFast]          │    │
│  │                                                         │    │
│  │  5. Return answer với citations                         │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           ↓                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  RESPONSE                                               │    │
│  │                                                         │    │
│  │  "VinFast nhận nhiều đánh giá tiêu cực về:             │    │
│  │   1. PIN (45% negative) - Sụt nhanh, dung lượng thấp   │    │
│  │   2. GIÁ (30% negative) - Đắt so với đối thủ           │    │
│  │   3. DỊCH VỤ (15% negative) - Chậm trễ, thiếu phụ tùng"│    │
│  │                                                         │    │
│  │  Citations:                                             │    │
│  │  [1] "Xe đẹp nhưng pin yếu" - Nguyễn A, 01/02/2026     │    │
│  │  [2] "Giá hơi cao" - Trần B, 02/02/2026                │    │
│  │  ...                                                    │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

---


## PHẦN 4: SETUP RAG ĐỠN GIẢN - 2 MODELS

### 4.1 Mô hình đơn giản nhất

Bạn đúng rồi! RAG cơ bản chỉ cần 2 models:

```
┌─────────────────────────────────────────────────────────────────┐
│                    RAG = 2 MODELS                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  MODEL 1: EMBEDDING MODEL                                       │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Vai trò: Convert text → vector (array of numbers)     │    │
│  │                                                         │    │
│  │  Input:  "Xe đẹp nhưng pin yếu"                        │    │
│  │  Output: [0.123, -0.456, 0.789, ..., 0.321]           │    │
│  │          (1536 numbers)                                 │    │
│  │                                                         │    │
│  │  Options:                                               │    │
│  │  ├── OpenAI text-embedding-3-small (API, $$$)          │    │
│  │  ├── sentence-transformers (local, free)               │    │
│  │  └── PhoBERT (Vietnamese, local, free)                 │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
│  MODEL 2: GENERATION MODEL (LLM)                                │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  Vai trò: Đọc context + Generate answer                │    │
│  │                                                         │    │
│  │  Input:  Context: [10 relevant documents]              │    │
│  │          Question: "VinFast bị đánh giá tiêu cực về gì?"│   │
│  │                                                         │    │
│  │  Output: "VinFast nhận nhiều đánh giá tiêu cực về..."  │    │
│  │                                                         │    │
│  │  Options:                                               │    │
│  │  ├── OpenAI GPT-4 (API, $$$, best quality)             │    │
│  │  ├── OpenAI GPT-3.5-turbo (API, $$, good)              │    │
│  │  ├── Llama 3 (local, free, need GPU)                   │    │
│  │  └── Gemini Pro (API, $$, good)                        │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 Code Setup Đơn Giản (Go)

#### Setup 1: Embedding Model (OpenAI)

```go
// pkg/embedding/openai.go
package embedding

import (
    "context"
    openai "github.com/sashabaranov/go-openai"
)

type Embedder struct {
    client *openai.Client
}

func NewEmbedder(apiKey string) *Embedder {
    return &Embedder{
        client: openai.NewClient(apiKey),
    }
}

// Hàm duy nhất: Text → Vector
func (e *Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
    resp, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
        Model: openai.SmallEmbedding3,  // text-embedding-3-small
        Input: []string{text},
    })
    if err != nil {
        return nil, err
    }
    
    return resp.Data[0].Embedding, nil  // 1536 numbers
}
```

**Usage:**
```go
embedder := embedding.NewEmbedder("sk-...")
vector, _ := embedder.Embed(ctx, "Xe đẹp nhưng pin yếu")
// vector = [0.123, -0.456, ..., 0.321] (1536 floats)
```

#### Setup 2: Generation Model (OpenAI GPT-4)

```go
// pkg/llm/openai.go
package llm

import (
    "context"
    "fmt"
    openai "github.com/sashabaranov/go-openai"
)

type Generator struct {
    client *openai.Client
}

func NewGenerator(apiKey string) *Generator {
    return &Generator{
        client: openai.NewClient(apiKey),
    }
}

// Hàm duy nhất: Context + Question → Answer
func (g *Generator) Generate(ctx context.Context, question string, context []string) (string, error) {
    // 1. Build context text
    contextText := ""
    for i, doc := range context {
        contextText += fmt.Sprintf("[%d] %s\n\n", i+1, doc)
    }
    
    // 2. Build prompt
    prompt := fmt.Sprintf(`Bạn là trợ lý phân tích dữ liệu.

Context (dữ liệu từ database):
%s

Câu hỏi: %s

Hãy trả lời dựa trên context trên. Trích dẫn nguồn bằng [1], [2]...`, contextText, question)
    
    // 3. Call GPT-4
    resp, err := g.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model: openai.GPT4,
        Messages: []openai.ChatCompletionMessage{
            {
                Role:    openai.ChatMessageRoleUser,
                Content: prompt,
            },
        },
        Temperature: 0.7,
    })
    
    if err != nil {
        return "", err
    }
    
    return resp.Choices[0].Message.Content, nil
}
```

**Usage:**
```go
generator := llm.NewGenerator("sk-...")

context := []string{
    "Xe đẹp nhưng pin yếu",
    "Giá hơi cao",
    "Dịch vụ chậm",
}

answer, _ := generator.Generate(ctx, "VinFast bị đánh giá tiêu cực về gì?", context)
// answer = "VinFast nhận đánh giá tiêu cực về PIN [1], GIÁ [2], và DỊCH VỤ [3]..."
```

### 4.3 RAG Pipeline Hoàn Chỉnh

```go
// internal/rag/pipeline.go
package rag

type Pipeline struct {
    embedder  *embedding.Embedder
    qdrant    *qdrant.Client
    generator *llm.Generator
}

func (p *Pipeline) Query(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
    // BƯỚC 1: Embed câu hỏi
    queryVector, err := p.embedder.Embed(ctx, req.Question)
    if err != nil {
        return nil, err
    }
    
    // BƯỚC 2: Search trong Qdrant
    searchResult, err := p.qdrant.Search(ctx, &qdrant.SearchPoints{
        CollectionName: "smap_analytics",
        Vector:         queryVector,
        Filter: &qdrant.Filter{
            Must: []*qdrant.Condition{
                {
                    Field: &qdrant.FieldCondition{
                        Key: "project_id",
                        Match: &qdrant.Match{
                            Keywords: &qdrant.RepeatedStrings{
                                Strings: req.ProjectIDs,  // Campaign scope
                            },
                        },
                    },
                },
            },
        },
        Limit: 10,
    })
    if err != nil {
        return nil, err
    }
    
    // BƯỚC 3: Extract context từ search results
    context := []string{}
    for _, point := range searchResult {
        content := point.Payload["content"].GetStringValue()
        context = append(context, content)
    }
    
    // BƯỚC 4: Generate answer
    answer, err := p.generator.Generate(ctx, req.Question, context)
    if err != nil {
        return nil, err
    }
    
    return &QueryResponse{
        Answer:    answer,
        Citations: searchResult,
    }, nil
}
```

**Sử dụng:**
```go
pipeline := rag.NewPipeline(embedder, qdrantClient, generator)

resp, _ := pipeline.Query(ctx, &rag.QueryRequest{
    Question:   "VinFast bị đánh giá tiêu cực về gì?",
    ProjectIDs: []string{"proj_vf8"},  // Campaign scope
})

fmt.Println(resp.Answer)
// "VinFast nhận đánh giá tiêu cực về PIN, GIÁ, và DỊCH VỤ..."
```

### 4.4 Tóm tắt: Chỉ cần 2 models!

```
┌─────────────────────────────────────────────────────────────────┐
│                    RAG WORKFLOW                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  User Question: "VinFast bị đánh giá tiêu cực về gì?"           │
│       │                                                         │
│       ↓                                                         │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  MODEL 1: Embedding                                     │    │
│  │  Question → Vector [0.123, -0.456, ...]                │    │
│  └─────────────────────────────────────────────────────────┘    │
│       │                                                         │
│       ↓                                                         │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  QDRANT: Vector Search                                  │    │
│  │  Find similar vectors → Top 10 documents                │    │
│  │  Filter: project_id IN campaign.projects                │    │
│  └─────────────────────────────────────────────────────────┘    │
│       │                                                         │
│       ↓                                                         │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  MODEL 2: Generation (LLM)                              │    │
│  │  Context + Question → Answer                            │    │
│  └─────────────────────────────────────────────────────────┘    │
│       │                                                         │
│       ↓                                                         │
│  Answer: "VinFast nhận đánh giá tiêu cực về PIN, GIÁ..."       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Không cần:**
- ❌ Fine-tuning models
- ❌ Training custom models
- ❌ Complex ML pipelines

**Chỉ cần:**
- ✅ OpenAI API key (hoặc local models)
- ✅ Qdrant vector database
- ✅ 2 API calls: Embedding + Generation

---

## PHẦN 5: CHECKLIST TRIỂN KHAI

### 5.1 Infrastructure Setup

```bash
# 1. Start Qdrant
docker run -d --name qdrant \
  -p 6333:6333 \
  -v $(pwd)/qdrant_storage:/qdrant/storage \
  qdrant/qdrant:latest

# 2. Create collection
curl -X PUT 'http://localhost:6333/collections/smap_analytics' \
  -H 'Content-Type: application/json' \
  -d '{
    "vectors": {
      "size": 1536,
      "distance": "Cosine"
    }
  }'

# 3. Verify
curl 'http://localhost:6333/collections/smap_analytics'
```

### 5.2 Code Implementation Checklist

- [ ] **Embedding Service**
  - [ ] OpenAI client setup
  - [ ] Embed() function
  - [ ] Error handling

- [ ] **Qdrant Client**
  - [ ] Connection setup
  - [ ] Upsert() function
  - [ ] Search() function với filters

- [ ] **LLM Service**
  - [ ] OpenAI client setup
  - [ ] Generate() function
  - [ ] Prompt engineering

- [ ] **RAG Pipeline**
  - [ ] Query() orchestration
  - [ ] Campaign scope filtering
  - [ ] Citation extraction

- [ ] **API Endpoints**
  - [ ] POST /knowledge/index (indexing)
  - [ ] POST /api/v1/chat (query)
  - [ ] GET /api/v1/chat/history

### 5.3 Data Flow Checklist

- [ ] **Ingest Service**
  - [ ] File upload endpoint
  - [ ] AI Schema Agent integration
  - [ ] UAP transformation

- [ ] **Analytics Service**
  - [ ] Kafka consumer
  - [ ] Sentiment analysis
  - [ ] Aspect extraction
  - [ ] Save to PostgreSQL

- [ ] **Knowledge Service**
  - [ ] Listen for analytics completion
  - [ ] Generate embeddings
  - [ ] Index to Qdrant
  - [ ] Query endpoint

### 5.4 Testing Checklist

```bash
# Test 1: Index a document
curl -X POST http://localhost:8080/knowledge/index \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "proj_vf8",
    "content": "Xe đẹp nhưng pin yếu",
    "sentiment": "NEGATIVE",
    "aspects": [{"aspect": "PIN", "sentiment": "NEGATIVE"}]
  }'

# Test 2: Query
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "campaign_id": "camp_001",
    "message": "VinFast bị đánh giá tiêu cực về gì?"
  }'

# Test 3: Verify Qdrant
curl 'http://localhost:6333/collections/smap_analytics/points/scroll' \
  -H "Content-Type: application/json" \
  -d '{
    "limit": 10,
    "with_payload": true,
    "with_vector": false
  }'
```

---

## PHẦN 6: TÓM TẮT - GIẢI ĐÁP 3 CÂU HỎI

### ❓ Câu hỏi 1: RAG gắn với Campaign như thế nào?

**Trả lời:**

1. Campaign chứa nhiều Projects (many-to-many relationship)
2. Mỗi Project có nhiều Data Sources
3. Data từ tất cả sources được transform → UAP → Analytics → Vector
4. Vector được lưu trong Qdrant với `project_id` trong payload
5. Khi query trong Campaign, RAG filter: `project_id IN campaign.project_ids`
6. **Kết quả:** RAG CHỈ trả lời dựa trên data từ projects trong campaign đó

**Code:**
```go
// Get campaign projects
projectIDs := getCampaignProjects(campaignID)  // ["proj_vf8", "proj_byd"]

// Search with filter
qdrant.Search(ctx, &qdrant.SearchPoints{
    Filter: &qdrant.Filter{
        Must: []*qdrant.Condition{
            {Field: "project_id", Match: projectIDs},  // CHỈ search trong campaign
        },
    },
})
```

### ❓ Câu hỏi 2: Flow data chi tiết từ analyst đến RAG?

**Trả lời:**

```
T0: Analyst upload file
    ↓ (5-10s)
T1: AI Schema Agent suggest mapping
    ↓ (user confirm)
T2: Transform raw → UAP (500 records)
    ↓ (publish to Kafka)
T3: Analytics Pipeline (Sentiment + Aspect)
    ↓ (1-5 phút, parallel processing)
T4: Save to PostgreSQL với labels
    ↓ (trigger indexing)
T5: Generate embeddings (OpenAI API)
    ↓ (10-30s)
T6: Upsert to Qdrant
    ↓
✅ RAG sẵn sàng trả lời
```

**Key Point:** Data CHỈ được index vào Qdrant SAU KHI có đủ labels (sentiment + aspects)

### ❓ Câu hỏi 3: Setup RAG đơn giản với 2 models?

**Trả lời:**

**Đúng rồi! Chỉ cần 2 models:**

1. **Embedding Model** (text → vector)
   - OpenAI text-embedding-3-small
   - Input: "Xe đẹp nhưng pin yếu"
   - Output: [0.123, -0.456, ..., 0.321] (1536 floats)

2. **Generation Model** (context + question → answer)
   - OpenAI GPT-4
   - Input: Context + Question
   - Output: Answer với citations

**Code tối giản:**
```go
// 1. Embed query
vector := embedder.Embed("VinFast bị đánh giá tiêu cực về gì?")

// 2. Search Qdrant
results := qdrant.Search(vector, filters)

// 3. Generate answer
answer := llm.Generate(question, results)
```

**Không cần:**
- ❌ Training models
- ❌ Fine-tuning
- ❌ Complex ML pipelines

**Chỉ cần:**
- ✅ OpenAI API key
- ✅ Qdrant running
- ✅ 3 API calls

---

## PHẦN 7: NEXT STEPS

### Bước tiếp theo để implement:

1. **Setup Infrastructure (1 giờ)**
   - Start Qdrant container
   - Create collection
   - Get OpenAI API key

2. **Implement Embedding Service (2 giờ)**
   - OpenAI client
   - Embed() function
   - Unit tests

3. **Implement Qdrant Client (2 giờ)**
   - Connection setup
   - Upsert() function
   - Search() function

4. **Implement LLM Service (2 giờ)**
   - OpenAI client
   - Generate() function
   - Prompt engineering

5. **Implement RAG Pipeline (4 giờ)**
   - Query() orchestration
   - Campaign scope filtering
   - Citation extraction

6. **Integration Testing (2 giờ)**
   - End-to-end flow
   - Performance testing
   - Error handling

**Total: ~13 giờ để có RAG working prototype**

---

## TÀI LIỆU THAM KHẢO

1. **Qdrant Docs:** https://qdrant.tech/documentation/
2. **OpenAI Embeddings:** https://platform.openai.com/docs/guides/embeddings
3. **OpenAI Chat Completions:** https://platform.openai.com/docs/guides/chat
4. **go-openai:** https://github.com/sashabaranov/go-openai
5. **go-client (Qdrant):** https://github.com/qdrant/go-client

---

**Hy vọng document này giải đáp được các thắc mắc của bạn! 🚀**
