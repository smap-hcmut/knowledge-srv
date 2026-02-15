# Hướng Dẫn Dựng RAG System Cơ Bản cho SMAP

**Tài liệu tham khảo:** `documents/planning/migration-plan-v2.md`  
**Ngày tạo:** 15/02/2026  
**Mục đích:** Hướng dẫn triển khai RAG (Retrieval-Augmented Generation) cho hệ thống SMAP On-Premise

---

## 1. TỔNG QUAN RAG TRONG SMAP

### 1.1 RAG là gì?

RAG (Retrieval-Augmented Generation) là kỹ thuật kết hợp:

- **Retrieval:** Tìm kiếm thông tin liên quan từ database
- **Generation:** Sử dụng LLM để tạo câu trả lời dựa trên thông tin tìm được

### 1.2 Vai trò trong SMAP

RAG được sử dụng trong **UC-03: Campaign War Room** - cho phép người dùng:

- Hỏi đáp bằng ngôn ngữ tự nhiên về dữ liệu phân tích
- So sánh cross-brand (VD: "So sánh VinFast vs BYD về giá")
- Tạo báo cáo tự động từ dữ liệu

### 1.3 Kiến trúc RAG trong SMAP

```
┌─────────────────────────────────────────────────────────────────┐
│                    RAG ARCHITECTURE                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  User Query: "Tại sao VinFast bị đánh giá tiêu cực về pin?"     │
│       │                                                         │
│       ↓                                                         │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │           KNOWLEDGE SERVICE (Go)                        │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │ 1. Query Processing                             │    │    │
│  │  │    - Extract intent: "negative sentiment"       │    │    │
│  │  │    - Extract entities: "VinFast", "pin"         │    │    │
│  │  │    - Get campaign context: project_ids          │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  │                      ↓                                  │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │ 2. Embedding Generation                         │    │    │
│  │  │    - Convert query → vector (OpenAI API)        │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  │                      ↓                                  │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │ 3. Hybrid Search (Qdrant)                       │    │    │
│  │  │    - Vector similarity search                   │    │    │
│  │  │    - Filter: project_id IN campaign.projects    │    │    │
│  │  │    - Filter: sentiment = "NEGATIVE"             │    │    │
│  │  │    - Filter: aspects CONTAINS "PIN"             │    │    │
│  │  │    → Top 10 relevant records                    │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  │                      ↓                                  │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │ 4. Context Building                             │    │    │
│  │  │    - Aggregate retrieved records                │    │    │
│  │  │    - Add metadata (author, date, platform)      │    │    │
│  │  │    - Format for LLM prompt                      │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  │                      ↓                                  │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │ 5. Answer Generation (LLM)                      │    │    │
│  │  │    - Send context + query to OpenAI             │    │    │
│  │  │    - Generate answer with citations             │    │    │
│  │  │    - Extract key insights                       │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                      ↓                                          │
│  Response: "VinFast nhận 45% đánh giá tiêu cực về pin vì..."    │
│  Citations: [Post #123, Post #456, Post #789]             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. TECH STACK

### 2.1 Core Components

| Component           | Technology                    | Vai trò                             |
| ------------------- | ----------------------------- | ----------------------------------- |
| **Vector Database** | Qdrant                        | Lưu trữ embeddings, semantic search |
| **Embedding Model** | OpenAI text-embedding-3-small | Convert text → vector               |
| **LLM**             | OpenAI GPT-4                  | Generate answers                    |
| **Backend**         | Golang                        | Knowledge Service API               |
| **Framework**       | LangChain (optional)          | RAG orchestration                   |

### 2.2 Data Flow

```
Raw Data (UAP)
    ↓
Analytics Pipeline (Sentiment + Aspect labeling)
    ↓
Vector Indexing (ONLY after labeling complete)
    ↓
Qdrant Vector DB
    ↓
RAG Query
```

**Lưu ý quan trọng:** Data chỉ được đưa vào Qdrant SAU KHI đã có đủ labels (sentiment + aspects) từ Analytics Pipeline.

---

## 3. SETUP QDRANT VECTOR DATABASE

### 3.1 Cài đặt Qdrant (Docker)

```bash
# Pull Qdrant image
docker pull qdrant/qdrant:latest

# Run Qdrant container
docker run -d \
  --name qdrant \
  -p 6333:6333 \
  -p 6334:6334 \
  -v $(pwd)/qdrant_storage:/qdrant/storage \
  qdrant/qdrant:latest
```

### 3.2 Tạo Collection

```bash
# Create collection với schema phù hợp SMAP
curl -X PUT 'http://localhost:6333/collections/smap_analytics' \
  -H 'Content-Type: application/json' \
  -d '{
    "vectors": {
      "size": 1536,
      "distance": "Cosine"
    },
    "optimizers_config": {
      "indexing_threshold": 10000
    }
  }'
```

### 3.3 Schema Vector Record

```json
{
  "id": "uuid-v4",
  "vector": [0.123, -0.456, ...],  // 1536 dimensions
  "payload": {
    "project_id": "proj_vinfast_vf8",
    "source_id": "src_tiktok_crawl",
    "content": "Xe đi êm nhưng pin sụt nhanh quá",
    "content_created_at": 1707206400,  // Unix timestamp
    "platform": "tiktok",
    "sentiment": "NEGATIVE",
    "sentiment_score": -0.75,
    "aspects": [
      {
        "aspect": "PIN",
        "sentiment": "NEGATIVE",
        "score": -0.8
      },
      {
        "aspect": "DESIGN",
        "sentiment": "POSITIVE",
        "score": 0.6
      }
    ],
    "keywords": ["pin", "sụt nhanh", "xe điện"],
    "metadata": {
      "author": "user123",
      "engagement": 1500
    }
  }
}
```

---

## 4. KNOWLEDGE SERVICE IMPLEMENTATION

### 4.1 Service Structure

```
knowledge-service/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── embedding/
│   │   └── openai.go          # OpenAI embedding client
│   ├── vectordb/
│   │   └── qdrant.go          # Qdrant client
│   ├── llm/
│   │   └── openai.go          # OpenAI LLM client
│   ├── rag/
│   │   ├── retriever.go       # Hybrid search logic
│   │   ├── generator.go       # Answer generation
│   │   └── pipeline.go        # RAG orchestration
│   └── api/
│       ├── chat.go            # Chat endpoints
│       └── index.go           # Indexing endpoints
├── pkg/
│   └── models/
│       └── types.go           # Data structures
└── config/
    └── config.yaml
```

### 4.2 Core Functions

#### 4.2.1 Embedding Generation

```go
// internal/embedding/openai.go
package embedding

import (
    "context"
    openai "github.com/sashabaranov/go-openai"
)

type OpenAIEmbedder struct {
    client *openai.Client
}

func NewOpenAIEmbedder(apiKey string) *OpenAIEmbedder {
    return &OpenAIEmbedder{
        client: openai.NewClient(apiKey),
    }
}

func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
    resp, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
        Model: openai.SmallEmbedding3,
        Input: []string{text},
    })
    if err != nil {
        return nil, err
    }

    return resp.Data[0].Embedding, nil
}
```

#### 4.2.2 Vector Indexing

```go
// internal/api/index.go
package api

import (
    "context"
    "github.com/google/uuid"
    "github.com/qdrant/go-client/qdrant"
)

type IndexRequest struct {
    ProjectID        string                 `json:"project_id"`
    SourceID         string                 `json:"source_id"`
    Content          string                 `json:"content"`
    ContentCreatedAt int64                  `json:"content_created_at"`
    Platform         string                 `json:"platform"`
    Sentiment        string                 `json:"sentiment"`
    SentimentScore   float64                `json:"sentiment_score"`
    Aspects          []Aspect               `json:"aspects"`
    Keywords         []string               `json:"keywords"`
    Metadata         map[string]interface{} `json:"metadata"`
}

func (h *Handler) IndexDocument(ctx context.Context, req *IndexRequest) error {
    // 1. Generate embedding
    vector, err := h.embedder.Embed(ctx, req.Content)
    if err != nil {
        return err
    }

    // 2. Prepare Qdrant point
    point := &qdrant.PointStruct{
        Id: qdrant.NewIDUUID(uuid.New().String()),
        Vectors: qdrant.NewVectors(vector...),
        Payload: qdrant.NewValueMap(map[string]interface{}{
            "project_id":         req.ProjectID,
            "source_id":          req.SourceID,
            "content":            req.Content,
            "content_created_at": req.ContentCreatedAt,
            "platform":           req.Platform,
            "sentiment":          req.Sentiment,
            "sentiment_score":    req.SentimentScore,
            "aspects":            req.Aspects,
            "keywords":           req.Keywords,
            "metadata":           req.Metadata,
        }),
    }

    // 3. Upsert to Qdrant
    _, err = h.qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
        CollectionName: "smap_analytics",
        Points:         []*qdrant.PointStruct{point},
    })

    return err
}
```

#### 4.2.3 Hybrid Search

```go
// internal/rag/retriever.go
package rag

import (
    "context"
    "github.com/qdrant/go-client/qdrant"
)

type SearchRequest struct {
    Query       string   `json:"query"`
    CampaignID  string   `json:"campaign_id"`
    ProjectIDs  []string `json:"project_ids"`  // From campaign
    Sentiment   string   `json:"sentiment,omitempty"`
    Aspects     []string `json:"aspects,omitempty"`
    DateFrom    int64    `json:"date_from,omitempty"`
    DateTo      int64    `json:"date_to,omitempty"`
    Limit       int      `json:"limit"`
}

func (r *Retriever) Search(ctx context.Context, req *SearchRequest) ([]*SearchResult, error) {
    // 1. Generate query embedding
    queryVector, err := r.embedder.Embed(ctx, req.Query)
    if err != nil {
        return nil, err
    }

    // 2. Build filters
    filters := &qdrant.Filter{
        Must: []*qdrant.Condition{
            // Filter by campaign projects
            {
                ConditionOneOf: &qdrant.Condition_Field{
                    Field: &qdrant.FieldCondition{
                        Key: "project_id",
                        Match: &qdrant.Match{
                            MatchValue: &qdrant.Match_Keywords{
                                Keywords: &qdrant.RepeatedStrings{
                                    Strings: req.ProjectIDs,
                                },
                            },
                        },
                    },
                },
            },
        },
    }

    // Optional: Filter by sentiment
    if req.Sentiment != "" {
        filters.Must = append(filters.Must, &qdrant.Condition{
            ConditionOneOf: &qdrant.Condition_Field{
                Field: &qdrant.FieldCondition{
                    Key: "sentiment",
                    Match: &qdrant.Match{
                        MatchValue: &qdrant.Match_Keyword{
                            Keyword: req.Sentiment,
                        },
                    },
                },
            },
        })
    }

    // Optional: Filter by date range
    if req.DateFrom > 0 || req.DateTo > 0 {
        filters.Must = append(filters.Must, &qdrant.Condition{
            ConditionOneOf: &qdrant.Condition_Field{
                Field: &qdrant.FieldCondition{
                    Key: "content_created_at",
                    Range: &qdrant.Range{
                        Gte: float64(req.DateFrom),
                        Lte: float64(req.DateTo),
                    },
                },
            },
        })
    }

    // 3. Execute search
    searchResult, err := r.qdrantClient.Search(ctx, &qdrant.SearchPoints{
        CollectionName: "smap_analytics",
        Vector:         queryVector,
        Filter:         filters,
        Limit:          uint64(req.Limit),
        WithPayload:    &qdrant.WithPayloadSelector{
            SelectorOptions: &qdrant.WithPayloadSelector_Enable{
                Enable: true,
            },
        },
    })

    if err != nil {
        return nil, err
    }

    // 4. Convert to SearchResult
    results := make([]*SearchResult, len(searchResult))
    for i, point := range searchResult {
        results[i] = &SearchResult{
            Score:   point.Score,
            Content: point.Payload["content"].GetStringValue(),
            // ... map other fields
        }
    }

    return results, nil
}
```

#### 4.2.4 Answer Generation

```go
// internal/rag/generator.go
package rag

import (
    "context"
    "fmt"
    openai "github.com/sashabaranov/go-openai"
)

func (g *Generator) Generate(ctx context.Context, query string, context []*SearchResult) (string, error) {
    // 1. Build context from search results
    contextText := ""
    for i, result := range context {
        contextText += fmt.Sprintf("[%d] %s\n", i+1, result.Content)
    }

    // 2. Build prompt
    prompt := fmt.Sprintf(`Bạn là trợ lý phân tích dữ liệu cho SMAP.

Context (dữ liệu phân tích):
%s

Câu hỏi: %s

Hãy trả lời câu hỏi dựa trên context trên. Trích dẫn nguồn bằng số [1], [2]...
Nếu không tìm thấy thông tin, hãy nói rõ.`, contextText, query)

    // 3. Call OpenAI
    resp, err := g.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model: openai.GPT4,
        Messages: []openai.ChatCompletionMessage{
            {
                Role:    openai.ChatMessageRoleSystem,
                Content: "Bạn là trợ lý phân tích dữ liệu chuyên nghiệp.",
            },
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

---

## 5. CAMPAIGN-SCOPED RAG

### 5.1 Campaign Context

Campaign là đơn vị phân tích logic (Tầng 3) chứa nhiều Projects:

```
Campaign "So sánh Xe điện"
├── Project "Monitor VF8" (brand=VinFast)
└── Project "Monitor BYD Seal" (brand=BYD)
```

### 5.2 Query Flow

```go
// internal/api/chat.go
package api

func (h *Handler) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
    // 1. Get campaign projects
    campaign, err := h.projectService.GetCampaign(ctx, req.CampaignID)
    if err != nil {
        return nil, err
    }

    projectIDs := campaign.ProjectIDs

    // 2. Search with campaign scope
    searchReq := &rag.SearchRequest{
        Query:      req.Message,
        CampaignID: req.CampaignID,
        ProjectIDs: projectIDs,  // Filter by campaign projects
        Limit:      10,
    }

    results, err := h.retriever.Search(ctx, searchReq)
    if err != nil {
        return nil, err
    }

    // 3. Generate answer
    answer, err := h.generator.Generate(ctx, req.Message, results)
    if err != nil {
        return nil, err
    }

    return &ChatResponse{
        Answer:    answer,
        Citations: results,
    }, nil
}
```

---

## 6. DEPLOYMENT

### 6.1 Docker Compose

```yaml
version: "3.8"

services:
  qdrant:
    image: qdrant/qdrant:latest
    ports:
      - "6333:6333"
      - "6334:6334"
    volumes:
      - qdrant_storage:/qdrant/storage
    environment:
      - QDRANT__SERVICE__GRPC_PORT=6334

  knowledge-service:
    build: ./knowledge-service
    ports:
      - "8080:8080"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - QDRANT_URL=http://qdrant:6333
      - POSTGRES_URL=${POSTGRES_URL}
    depends_on:
      - qdrant

volumes:
  qdrant_storage:
```

### 6.2 Kubernetes (Helm)

```yaml
# charts/knowledge-service/values.yaml
replicaCount: 2

image:
  repository: smap/knowledge-service
  tag: "1.0.0"

env:
  - name: OPENAI_API_KEY
    valueFrom:
      secretKeyRef:
        name: openai-secret
        key: api-key
  - name: QDRANT_URL
    value: "http://qdrant:6333"

resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    memory: "1Gi"
    cpu: "1000m"
```

---

## 7. TESTING & OPTIMIZATION

### 7.1 Test Cases

```bash
# Test 1: Simple query
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "campaign_id": "campaign_123",
    "message": "VinFast được đánh giá như thế nào về pin?"
  }'

# Test 2: Comparison query
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "campaign_id": "campaign_123",
    "message": "So sánh VinFast và BYD về giá cả"
  }'

# Test 3: Temporal query
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{
    "campaign_id": "campaign_123",
    "message": "Xu hướng đánh giá về VinFast tuần này"
  }'
```

### 7.2 Performance Optimization

| Aspect              | Strategy                                          |
| ------------------- | ------------------------------------------------- |
| **Embedding Cache** | Cache embeddings của queries phổ biến trong Redis |
| **Vector Index**    | Qdrant HNSW index với `m=16, ef_construct=100`    |
| **Batch Indexing**  | Index theo batch 100-500 records                  |
| **Query Rewrite**   | Normalize queries trước khi embed                 |
| **Result Caching**  | Cache kết quả search 5-10 phút                    |

---

## 8. MONITORING & METRICS

### 8.1 Key Metrics

```go
// Prometheus metrics
var (
    searchLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name: "rag_search_latency_seconds",
        Help: "RAG search latency",
    })

    generationLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name: "rag_generation_latency_seconds",
        Help: "Answer generation latency",
    })

    relevanceScore = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "rag_relevance_score",
        Help: "Average relevance score",
    })
)
```

### 8.2 Success Criteria

| Metric                     | Target               |
| -------------------------- | -------------------- |
| Search latency             | < 500ms              |
| Generation latency         | < 2s                 |
| Answer relevance           | > 70% (user rating)  |
| Campaign query performance | < 2s (cross-project) |

---

## 9. TROUBLESHOOTING

### 9.1 Common Issues

**Issue 1: Qdrant connection failed**

```bash
# Check Qdrant status
curl http://localhost:6333/health

# Check logs
docker logs qdrant
```

**Issue 2: Low relevance scores**

```
Solutions:
- Tune search filters (sentiment, aspects)
- Increase search limit (10 → 20)
- Improve prompt engineering
- Add query rewriting
```

**Issue 3: Slow queries**

```
Solutions:
- Enable Qdrant indexing
- Add Redis caching
- Optimize filters
- Use smaller embedding model
```

---

## 10. NEXT STEPS

### 10.1 MVP Features (Tuần 9-11)

- ✅ Basic RAG pipeline
- ✅ Campaign-scoped search
- ✅ Simple chat interface

### 10.2 Advanced Features (Phase 2)

- [ ] Multi-turn conversation (chat history)
- [ ] Smart suggestions based on visual data
- [ ] Report generation (PDF export)
- [ ] Artifact editing (inline + Google Docs)

### 10.3 Production Readiness

- [ ] Load testing (1000 concurrent users)
- [ ] Monitoring & alerting
- [ ] Backup & disaster recovery
- [ ] Security audit

---

## 11. API INTERFACE

### 11.1 Public Endpoints

Knowledge Service expose các API endpoints sau:

```
POST   /api/v1/chat                      # Chat với campaign
GET    /api/v1/conversations/{id}        # Lấy lịch sử chat
POST   /api/v1/search                    # Semantic search
POST   /api/v1/reports/generate          # Tạo báo cáo
GET    /api/v1/reports/{id}              # Check report status
GET    /api/v1/campaigns/{id}/suggestions # Smart suggestions
GET    /health                           # Health check
```

### 11.2 Internal Endpoints

```
POST   /internal/index                   # Index single document
POST   /internal/index/batch             # Batch indexing
```

### 11.3 Example: Chat API

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "campaign_id": "camp_001",
    "message": "VinFast bị đánh giá tiêu cực về gì?",
    "filters": {
      "sentiment": "NEGATIVE",
      "date_from": "2026-02-01T00:00:00Z"
    }
  }'
```

**Response:**
```json
{
  "answer": "VinFast nhận nhiều đánh giá tiêu cực về:\n1. PIN (45%) - Sụt nhanh...\n2. GIÁ (30%) - Đắt so với đối thủ...",
  "citations": [
    {
      "content": "Xe đẹp nhưng pin sụt nhanh",
      "source": {
        "platform": "tiktok",
        "author": "nguyen_van_a",
        "created_at": "2026-02-10T14:30:00Z"
      },
      "relevance_score": 0.92
    }
  ],
  "suggestions": [
    "Có bao nhiêu người phàn nàn về pin?",
    "So sánh pin VinFast với BYD"
  ]
}
```

**Chi tiết đầy đủ:** Xem `documents/input-output/KNOWLEDGE_SERVICE_API_SPEC.md`

---

## TÀI LIỆU THAM KHẢO

1. **Qdrant Documentation:** https://qdrant.tech/documentation/
2. **OpenAI Embeddings:** https://platform.openai.com/docs/guides/embeddings
3. **LangChain Go:** https://github.com/tmc/langchaingo
4. **Migration Plan v2.11:** `documents/planning/migration-plan-v2.md`
5. **API Specification:** `documents/input-output/KNOWLEDGE_SERVICE_API_SPEC.md`
6. **Data Flow:** `documents/RAG_DATA_FLOW_DETAILED.md`
7. **Analytics Schema:** `documents/input-output/ANALYTICS_SCHEMA_EXPLAINED.md`

---

**Lưu ý:** Đây là hướng dẫn cơ bản. Để triển khai production, cần bổ sung:

- Security (authentication, rate limiting)
- Scalability (horizontal scaling, load balancing)
- Observability (logging, tracing, metrics)
- Cost optimization (embedding caching, model selection)
