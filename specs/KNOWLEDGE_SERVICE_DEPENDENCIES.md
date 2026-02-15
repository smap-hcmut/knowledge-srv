# Knowledge Service Dependencies - Tất cả kết nối cần thiết

**Câu hỏi:** Knowledge Service chỉ cần kết nối với Qdrant thôi đúng không?  
**Trả lời:** KHÔNG! Cần nhiều dependencies khác.

---

## 1. TỔNG QUAN DEPENDENCIES

```
┌─────────────────────────────────────────────────────────────────┐
│                    KNOWLEDGE SERVICE                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  EXTERNAL DEPENDENCIES (Bắt buộc)                        │   │
│  ├──────────────────────────────────────────────────────────┤   │
│  │  1. Qdrant          - Vector database (search)           │   │
│  │  2. OpenAI API      - Embedding + LLM generation         │   │
│  │  3. PostgreSQL      - Metadata, conversation history     │   │
│  │  4. Redis           - Caching, rate limiting             │   │
│  │  5. Project Service - Get campaign projects              │   │
│  │  6. MinIO           - Report storage (PDF/DOCX)          │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  OPTIONAL DEPENDENCIES                                   │   │
│  ├──────────────────────────────────────────────────────────┤   │
│  │  7. Kafka           - Event publishing (optional)        │   │
│  │  8. Prometheus      - Metrics export (monitoring)        │   │
│  │  9. Jaeger          - Distributed tracing (observability)│   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. CHI TIẾT TỪNG DEPENDENCY

### 2.1 Qdrant (Vector Database) ⭐ BẮT BUỘC

**Vai trò:** Lưu trữ và search vectors

**Kết nối:**
```go
import "github.com/qdrant/go-client/qdrant"

client, err := qdrant.NewClient(&qdrant.Config{
    Host: "qdrant:6333",
    APIKey: os.Getenv("QDRANT_API_KEY"),
})
```

**Sử dụng cho:**
- ✅ Upsert vectors khi index documents
- ✅ Semantic search với filters
- ✅ Scroll/scan all vectors
- ✅ Delete vectors khi xóa data

**Operations:**
```go
// 1. Upsert vector
client.Upsert(ctx, &qdrant.UpsertPoints{
    CollectionName: "smap_analytics",
    Points: []*qdrant.PointStruct{...},
})

// 2. Search
client.Search(ctx, &qdrant.SearchPoints{
    CollectionName: "smap_analytics",
    Vector: queryVector,
    Filter: filters,
    Limit: 10,
})

// 3. Delete
client.Delete(ctx, &qdrant.DeletePoints{
    CollectionName: "smap_analytics",
    PointsSelector: &qdrant.PointsSelector{...},
})
```

**Config:**
```yaml
qdrant:
  url: "http://qdrant:6333"
  collection: "smap_analytics"
  api_key: "${QDRANT_API_KEY}"
  timeout: 30s
```

---

### 2.2 OpenAI API ⭐ BẮT BUỘC

**Vai trò:** Generate embeddings + LLM responses

**Kết nối:**
```go
import openai "github.com/sashabaranov/go-openai"

client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
```

**Sử dụng cho:**

**A. Embedding Generation (text → vector)**
```go
resp, err := client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
    Model: openai.SmallEmbedding3,  // text-embedding-3-small
    Input: []string{content},
})
vector := resp.Data[0].Embedding  // 1536 floats
```

**B. Answer Generation (context + query → answer)**
```go
resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
    Model: openai.GPT4,
    Messages: []openai.ChatCompletionMessage{
        {Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
        {Role: openai.ChatMessageRoleUser, Content: userQuery},
    },
    Temperature: 0.7,
})
answer := resp.Choices[0].Message.Content
```

**Config:**
```yaml
openai:
  api_key: "${OPENAI_API_KEY}"
  embedding_model: "text-embedding-3-small"
  llm_model: "gpt-4"
  max_retries: 3
  timeout: 60s
```

**Cost Estimation:**
```
Embedding: $0.00002 / 1K tokens
GPT-4: $0.03 / 1K tokens (input), $0.06 / 1K tokens (output)

Example:
- 1000 documents indexed: ~$0.20
- 100 chat queries: ~$3-5
```

---

### 2.3 PostgreSQL ⭐ BẮT BUỘC

**Vai trò:** Lưu metadata, conversation history, reports metadata

**Kết nối:**
```go
import "github.com/jackc/pgx/v5/pgxpool"

pool, err := pgxpool.New(ctx, os.Getenv("POSTGRES_URL"))
```

**Sử dụng cho:**

**A. Conversation History**
```sql
CREATE TABLE knowledge.conversations (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE knowledge.messages (
    id UUID PRIMARY KEY,
    conversation_id UUID REFERENCES knowledge.conversations(id),
    role VARCHAR(20) NOT NULL,  -- 'user' | 'assistant'
    content TEXT NOT NULL,
    citations JSONB,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

**B. Reports Metadata**
```sql
CREATE TABLE knowledge.reports (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    report_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,  -- 'PROCESSING' | 'COMPLETED' | 'FAILED'
    file_path TEXT,  -- MinIO path
    file_url TEXT,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);
```

**C. Vector Index Metadata (tracking)**
```sql
CREATE TABLE knowledge.indexed_documents (
    analytics_id UUID PRIMARY KEY,
    project_id UUID NOT NULL,
    vector_id VARCHAR(255) NOT NULL,  -- Qdrant point ID
    indexed_at TIMESTAMPTZ DEFAULT NOW(),
    embedding_model VARCHAR(50),
    INDEX idx_project (project_id),
    INDEX idx_indexed_at (indexed_at)
);
```

**Config:**
```yaml
postgres:
  url: "postgresql://user:pass@postgres:5432/smap"
  schema: "knowledge"
  max_connections: 20
  idle_timeout: 5m
```

---

### 2.4 Redis ⭐ BẮT BUỘC

**Vai trò:** Caching, rate limiting, session management

**Kết nối:**
```go
import "github.com/redis/go-redis/v9"

client := redis.NewClient(&redis.Options{
    Addr: "redis:6379",
    DB: 1,  // DB 1 for knowledge service
})
```

**Sử dụng cho:**

**A. Embedding Cache**
```go
// Cache key: embedding:{content_hash}
key := fmt.Sprintf("embedding:%s", contentHash)
cached, err := client.Get(ctx, key).Result()
if err == redis.Nil {
    // Generate new embedding
    vector := generateEmbedding(content)
    // Cache for 7 days
    client.Set(ctx, key, vector, 7*24*time.Hour)
}
```

**B. Search Results Cache**
```go
// Cache key: search:{campaign_id}:{query_hash}:{filters_hash}
key := fmt.Sprintf("search:%s:%s:%s", campaignID, queryHash, filtersHash)
cached, err := client.Get(ctx, key).Result()
if err == redis.Nil {
    // Perform search
    results := searchQdrant(query, filters)
    // Cache for 5 minutes
    client.Set(ctx, key, results, 5*time.Minute)
}
```

**C. Rate Limiting**
```go
// Rate limit: 60 requests per minute per user
key := fmt.Sprintf("ratelimit:chat:%s", userID)
count, _ := client.Incr(ctx, key).Result()
if count == 1 {
    client.Expire(ctx, key, 1*time.Minute)
}
if count > 60 {
    return ErrRateLimitExceeded
}
```

**Config:**
```yaml
redis:
  url: "redis://redis:6379/1"
  cache:
    embedding_ttl: 168h  # 7 days
    search_ttl: 5m
  rate_limit:
    enabled: true
    requests_per_minute: 60
```

**Cache Hit Rate Target:** > 60%

---

### 2.5 Project Service ⭐ BẮT BUỘC

**Vai trò:** Lấy thông tin campaign và projects

**Kết nối:**
```go
type ProjectServiceClient struct {
    baseURL string
    client  *http.Client
}

func NewProjectServiceClient(baseURL string) *ProjectServiceClient {
    return &ProjectServiceClient{
        baseURL: baseURL,
        client: &http.Client{Timeout: 10 * time.Second},
    }
}
```

**Sử dụng cho:**

**A. Get Campaign Projects**
```go
// GET /api/v1/campaigns/{campaign_id}
func (c *ProjectServiceClient) GetCampaign(ctx context.Context, campaignID string) (*Campaign, error) {
    url := fmt.Sprintf("%s/api/v1/campaigns/%s", c.baseURL, campaignID)
    resp, err := c.client.Get(url)
    // ...
    return campaign, nil
}

// Response:
type Campaign struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    ProjectIDs  []string `json:"project_ids"`  // ["proj_vf8", "proj_byd"]
}
```

**B. Validate Project Access**
```go
// Check if user has access to project
func (c *ProjectServiceClient) ValidateProjectAccess(ctx context.Context, userID, projectID string) (bool, error) {
    // Call Project Service API
}
```

**Config:**
```yaml
project_service:
  url: "http://project-service:8080"
  timeout: 10s
  retry:
    max_attempts: 3
    backoff: exponential
```

**Flow trong Chat:**
```
User: "VinFast bị đánh giá tiêu cực về gì?"
Campaign ID: "camp_001"
    ↓
Knowledge Service → Project Service
GET /api/v1/campaigns/camp_001
    ↓
Response: {project_ids: ["proj_vf8", "proj_byd"]}
    ↓
Search Qdrant với filter: project_id IN ["proj_vf8", "proj_byd"]
```

---

### 2.6 MinIO (Object Storage) ⭐ BẮT BUỘC (cho Report Generation)

**Vai trò:** Lưu trữ generated reports (PDF, DOCX, Markdown)

**Kết nối:**
```go
import "github.com/minio/minio-go/v7"

client, err := minio.New("minio:9000", &minio.Options{
    Creds: credentials.NewStaticV4(
        os.Getenv("MINIO_ACCESS_KEY"),
        os.Getenv("MINIO_SECRET_KEY"),
        "",
    ),
})
```

**Sử dụng cho:**

**A. Upload Report**
```go
func (s *ReportService) UploadReport(ctx context.Context, reportID string, data []byte) (string, error) {
    bucketName := "smap-reports"
    objectName := fmt.Sprintf("reports/%s.pdf", reportID)
    
    _, err := s.minioClient.PutObject(ctx, bucketName, objectName, 
        bytes.NewReader(data), int64(len(data)),
        minio.PutObjectOptions{ContentType: "application/pdf"},
    )
    
    // Generate presigned URL (valid for 7 days)
    url, err := s.minioClient.PresignedGetObject(ctx, bucketName, objectName, 
        7*24*time.Hour, nil)
    
    return url.String(), nil
}
```

**B. Download Report**
```go
func (s *ReportService) DownloadReport(ctx context.Context, reportID string) ([]byte, error) {
    bucketName := "smap-reports"
    objectName := fmt.Sprintf("reports/%s.pdf", reportID)
    
    object, err := s.minioClient.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
    defer object.Close()
    
    data, err := io.ReadAll(object)
    return data, nil
}
```

**Bucket Structure:**
```
smap-reports/
├── reports/
│   ├── report_001.pdf
│   ├── report_002.pdf
│   └── report_003.docx
└── artifacts/  (for Campaign Artifacts - future)
    ├── artifact_001.pdf
    └── artifact_002.md
```

**Config:**
```yaml
minio:
  endpoint: "minio:9000"
  access_key: "${MINIO_ACCESS_KEY}"
  secret_key: "${MINIO_SECRET_KEY}"
  bucket: "smap-reports"
  use_ssl: false
  presigned_url_expiry: 168h  # 7 days
```

**Report Generation Flow:**
```
User: POST /api/v1/reports/generate
    ↓
Knowledge Service:
    1. Search Qdrant → Get data
    2. Generate report content (LLM)
    3. Convert to PDF (library)
    4. Upload to MinIO
    5. Save metadata to PostgreSQL
    6. Return presigned URL
```

---

### 2.7 Kafka (Optional - Event Publishing)

**Vai trò:** Publish events cho monitoring, audit

**Kết nối:**
```go
import "github.com/segmentio/kafka-go"

writer := &kafka.Writer{
    Addr:     kafka.TCP("kafka:9092"),
    Topic:    "knowledge.events",
    Balancer: &kafka.LeastBytes{},
}
```

**Sử dụng cho:**

**A. Publish Events**
```go
// Event: Document indexed
writer.WriteMessages(ctx, kafka.Message{
    Key:   []byte(analyticsID),
    Value: []byte(`{"event": "DOCUMENT_INDEXED", "analytics_id": "...", "timestamp": "..."}`),
})

// Event: Chat query
writer.WriteMessages(ctx, kafka.Message{
    Key:   []byte(conversationID),
    Value: []byte(`{"event": "CHAT_QUERY", "campaign_id": "...", "query": "...", "timestamp": "..."}`),
})

// Event: Report generated
writer.WriteMessages(ctx, kafka.Message{
    Key:   []byte(reportID),
    Value: []byte(`{"event": "REPORT_GENERATED", "report_id": "...", "timestamp": "..."}`),
})
```

**Topics:**
```
knowledge.events           # All events
knowledge.indexed          # Document indexed events
knowledge.queries          # Chat query events
knowledge.reports          # Report generation events
```

**Config:**
```yaml
kafka:
  enabled: true  # Optional
  brokers:
    - kafka:9092
  topics:
    events: "knowledge.events"
```

---

### 2.8 Prometheus (Optional - Metrics)

**Vai trò:** Export metrics cho monitoring

**Kết nối:**
```go
import "github.com/prometheus/client_golang/prometheus"

// Define metrics
var (
    searchLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name: "knowledge_search_duration_seconds",
        Help: "Search latency",
    })
    
    llmCalls = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "knowledge_llm_calls_total",
        Help: "Total LLM calls",
    }, []string{"model", "status"})
)

// Register
prometheus.MustRegister(searchLatency, llmCalls)

// Expose endpoint
http.Handle("/metrics", promhttp.Handler())
```

**Metrics Exposed:**
```
# Search metrics
knowledge_search_duration_seconds
knowledge_search_results_count

# LLM metrics
knowledge_llm_calls_total{model="gpt-4", status="success"}
knowledge_llm_duration_seconds
knowledge_llm_tokens_used{type="prompt"}

# Cache metrics
knowledge_cache_hits_total
knowledge_cache_misses_total
```

---

### 2.9 Jaeger (Optional - Tracing)

**Vai trò:** Distributed tracing cho debugging

**Kết nối:**
```go
import "go.opentelemetry.io/otel"

tracer := otel.Tracer("knowledge-service")

// Trace request
ctx, span := tracer.Start(ctx, "ChatRequest")
defer span.End()

// Add attributes
span.SetAttributes(
    attribute.String("campaign_id", campaignID),
    attribute.String("query", query),
)
```

**Trace Example:**
```
ChatRequest (5.2s)
├── GetCampaign (50ms) → Project Service
├── GenerateEmbedding (800ms) → OpenAI API
├── SearchQdrant (200ms) → Qdrant
└── GenerateAnswer (4s) → OpenAI API
```

---

## 3. DEPENDENCY MATRIX

| Dependency | Bắt buộc? | Vai trò | Fallback nếu down |
|------------|-----------|---------|-------------------|
| **Qdrant** | ✅ YES | Vector search | ❌ Service unavailable |
| **OpenAI API** | ✅ YES | Embedding + LLM | ❌ Service unavailable |
| **PostgreSQL** | ✅ YES | Metadata, history | ❌ Service unavailable |
| **Redis** | ✅ YES | Caching, rate limit | ⚠️ Degraded (no cache) |
| **Project Service** | ✅ YES | Campaign info | ❌ Cannot resolve campaign |
| **MinIO** | ✅ YES* | Report storage | ⚠️ Reports unavailable |
| **Kafka** | ❌ NO | Event publishing | ✅ Continue without events |
| **Prometheus** | ❌ NO | Metrics | ✅ No monitoring |
| **Jaeger** | ❌ NO | Tracing | ✅ No tracing |

*MinIO chỉ bắt buộc nếu enable report generation feature

---

## 4. DOCKER COMPOSE SETUP

```yaml
version: '3.8'

services:
  knowledge-service:
    image: smap/knowledge-service:1.0.0
    ports:
      - "8080:8080"
    environment:
      # Qdrant
      - QDRANT_URL=http://qdrant:6333
      - QDRANT_API_KEY=${QDRANT_API_KEY}
      
      # OpenAI
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - OPENAI_EMBEDDING_MODEL=text-embedding-3-small
      - OPENAI_LLM_MODEL=gpt-4
      
      # PostgreSQL
      - POSTGRES_URL=postgresql://postgres:password@postgres:5432/smap
      
      # Redis
      - REDIS_URL=redis://redis:6379/1
      
      # Project Service
      - PROJECT_SERVICE_URL=http://project-service:8080
      
      # MinIO
      - MINIO_ENDPOINT=minio:9000
      - MINIO_ACCESS_KEY=${MINIO_ACCESS_KEY}
      - MINIO_SECRET_KEY=${MINIO_SECRET_KEY}
      
      # Optional: Kafka
      - KAFKA_ENABLED=true
      - KAFKA_BROKERS=kafka:9092
    depends_on:
      - qdrant
      - postgres
      - redis
      - minio
      - project-service
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  qdrant:
    image: qdrant/qdrant:latest
    ports:
      - "6333:6333"
    volumes:
      - qdrant_storage:/qdrant/storage

  postgres:
    image: postgres:15
    environment:
      - POSTGRES_DB=smap
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=password
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      - MINIO_ROOT_USER=${MINIO_ACCESS_KEY}
      - MINIO_ROOT_PASSWORD=${MINIO_SECRET_KEY}
    volumes:
      - minio_data:/data

  project-service:
    image: smap/project-service:1.0.0
    ports:
      - "8081:8080"

volumes:
  qdrant_storage:
  postgres_data:
  redis_data:
  minio_data:
```

---

## 5. ENVIRONMENT VARIABLES

```bash
# .env file
# Qdrant
QDRANT_URL=http://qdrant:6333
QDRANT_API_KEY=your-qdrant-api-key

# OpenAI
OPENAI_API_KEY=sk-your-openai-api-key

# PostgreSQL
POSTGRES_URL=postgresql://postgres:password@postgres:5432/smap

# Redis
REDIS_URL=redis://redis:6379/1

# Project Service
PROJECT_SERVICE_URL=http://project-service:8080

# MinIO
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin

# Optional: Kafka
KAFKA_ENABLED=true
KAFKA_BROKERS=kafka:9092
```

---

## 6. HEALTH CHECK DEPENDENCIES

```go
func (h *HealthHandler) CheckDependencies(ctx context.Context) *HealthResponse {
    resp := &HealthResponse{
        Status: "healthy",
        Dependencies: make(map[string]DependencyStatus),
    }
    
    // Check Qdrant
    if err := h.qdrantClient.HealthCheck(ctx); err != nil {
        resp.Dependencies["qdrant"] = DependencyStatus{
            Status: "unhealthy",
            Error: err.Error(),
        }
        resp.Status = "degraded"
    } else {
        resp.Dependencies["qdrant"] = DependencyStatus{Status: "healthy"}
    }
    
    // Check OpenAI (try embedding)
    if _, err := h.openaiClient.CreateEmbeddings(ctx, testRequest); err != nil {
        resp.Dependencies["openai"] = DependencyStatus{
            Status: "unhealthy",
            Error: err.Error(),
        }
        resp.Status = "degraded"
    } else {
        resp.Dependencies["openai"] = DependencyStatus{Status: "healthy"}
    }
    
    // Check PostgreSQL
    if err := h.db.Ping(ctx); err != nil {
        resp.Dependencies["postgres"] = DependencyStatus{
            Status: "unhealthy",
            Error: err.Error(),
        }
        resp.Status = "degraded"
    } else {
        resp.Dependencies["postgres"] = DependencyStatus{Status: "healthy"}
    }
    
    // Check Redis
    if err := h.redis.Ping(ctx).Err(); err != nil {
        resp.Dependencies["redis"] = DependencyStatus{
            Status: "unhealthy",
            Error: err.Error(),
        }
        // Redis down = degraded, not critical
    } else {
        resp.Dependencies["redis"] = DependencyStatus{Status: "healthy"}
    }
    
    // Check Project Service
    if _, err := h.projectClient.HealthCheck(ctx); err != nil {
        resp.Dependencies["project_service"] = DependencyStatus{
            Status: "unhealthy",
            Error: err.Error(),
        }
        resp.Status = "degraded"
    } else {
        resp.Dependencies["project_service"] = DependencyStatus{Status: "healthy"}
    }
    
    // Check MinIO
    if _, err := h.minioClient.BucketExists(ctx, "smap-reports"); err != nil {
        resp.Dependencies["minio"] = DependencyStatus{
            Status: "unhealthy",
            Error: err.Error(),
        }
        // MinIO down = reports unavailable, but chat still works
    } else {
        resp.Dependencies["minio"] = DependencyStatus{Status: "healthy"}
    }
    
    return resp
}
```

---

## 7. TÓM TẮT

### Dependencies BẮT BUỘC (6):

1. ✅ **Qdrant** - Vector search
2. ✅ **OpenAI API** - Embedding + LLM
3. ✅ **PostgreSQL** - Metadata, history
4. ✅ **Redis** - Caching, rate limiting
5. ✅ **Project Service** - Campaign info
6. ✅ **MinIO** - Report storage

### Dependencies OPTIONAL (3):

7. ⭕ **Kafka** - Event publishing
8. ⭕ **Prometheus** - Metrics
9. ⭕ **Jaeger** - Tracing

### Startup Order:

```
1. PostgreSQL (database)
2. Redis (cache)
3. Qdrant (vector DB)
4. MinIO (object storage)
5. Project Service (API dependency)
6. Knowledge Service (main service)
```

### Minimum Setup (Development):

```bash
# Start core dependencies
docker-compose up -d qdrant postgres redis minio

# Start Knowledge Service
go run cmd/server/main.go
```

### Full Setup (Production):

```bash
# Start all services
docker-compose up -d

# Verify health
curl http://localhost:8080/health/detailed
```

---

**Kết luận:** Knowledge Service KHÔNG CHỈ kết nối với Qdrant. Cần tối thiểu 6 dependencies để hoạt động đầy đủ!
