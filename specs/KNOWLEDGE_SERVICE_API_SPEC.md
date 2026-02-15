# Knowledge Service API Specification

**Service Name:** knowledge-service  
**Purpose:** RAG (Retrieval-Augmented Generation) cho Campaign-scoped Q&A  
**Port:** 8080  
**Base URL:** `/api/v1`

---

## 1. TỔNG QUAN

### 1.1 Vai trò của Knowledge Service

```
┌─────────────────────────────────────────────────────────────────┐
│                    KNOWLEDGE SERVICE                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  INPUT:                                                         │
│  ├── Analytics data (from analytics.post_analytics)             │
│  └── User queries (from Web UI)                                 │
│                                                                 │
│  PROCESSING:                                                    │
│  ├── Vector embedding (OpenAI)                                  │
│  ├── Semantic search (Qdrant)                                   │
│  ├── Context building                                           │
│  └── Answer generation (LLM)                                    │
│                                                                 │
│  OUTPUT:                                                        │
│  ├── Chat responses với citations                               │
│  ├── Generated reports (PDF/Markdown)                           │
│  └── Search results                                             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 Dependencies

**BẮT BUỘC (6):**
1. ✅ Qdrant - Vector database
2. ✅ OpenAI API - Embedding + LLM
3. ✅ PostgreSQL - Metadata, conversation history
4. ✅ Redis - Caching, rate limiting
5. ✅ Project Service - Campaign info
6. ✅ MinIO - Report storage

**OPTIONAL (3):**
7. ⭕ Kafka - Event publishing
8. ⭕ Prometheus - Metrics
9. ⭕ Jaeger - Tracing

**Chi tiết:** Xem `documents/input-output/KNOWLEDGE_SERVICE_DEPENDENCIES.md`

### 1.3 Core Capabilities

- ✅ **Indexing:** Nhận analytics data → Generate embeddings → Store in Qdrant
- ✅ **Search:** Semantic search với campaign scope filtering
- ✅ **Chat:** Conversational Q&A với context từ campaign
- ✅ **Report Generation:** Tạo báo cáo tự động từ data
- ✅ **Citation:** Trích dẫn nguồn chính xác

---

## 2. API ENDPOINTS

### 2.1 Health Check

```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "dependencies": {
    "qdrant": "connected",
    "openai": "available",
    "postgres": "connected"
  },
  "timestamp": "2026-02-15T10:30:00Z"
}
```

---


### 2.2 Index Document (Internal API)

**Endpoint:** `POST /internal/index`  
**Purpose:** Index analytics data vào Qdrant (được gọi bởi Analytics Service)  
**Authentication:** Internal service token

**Request:**
```json
{
  "analytics_id": "analytics_550e8400-e29b-41d4-a716-446655440000",
  "project_id": "proj_vinfast_vf8_monitor",
  "source_id": "src_tiktok_crawl_vinfast_vf8",
  "content": "Mình vừa lái thử VF8 được 2 tuần. Thiết kế xe rất đẹp...",
  "content_created_at": "2026-02-10T14:30:00Z",
  "platform": "tiktok",
  "overall_sentiment": "MIXED",
  "overall_sentiment_score": 0.15,
  "aspects": [
    {
      "aspect": "DESIGN",
      "sentiment": "POSITIVE",
      "sentiment_score": 0.85,
      "keywords": ["đẹp", "sang trọng"]
    },
    {
      "aspect": "BATTERY",
      "sentiment": "NEGATIVE",
      "sentiment_score": -0.72,
      "keywords": ["pin", "sụt nhanh"]
    }
  ],
  "keywords": ["VF8", "thiết kế", "pin", "giá"],
  "metadata": {
    "author": "nguyen_van_a_2024",
    "engagement": {
      "views": 45000,
      "likes": 3200
    }
  }
}
```

**Response:**
```json
{
  "success": true,
  "vector_id": "uap_001",
  "indexed_at": "2026-02-15T10:46:45Z",
  "processing_time_ms": 850
}
```

**Error Response:**
```json
{
  "success": false,
  "error": {
    "code": "EMBEDDING_FAILED",
    "message": "Failed to generate embedding from OpenAI",
    "details": "Rate limit exceeded"
  }
}
```

**Flow:**
```
Analytics Service (sau khi phân tích xong)
    ↓
POST /internal/index
    ↓
Knowledge Service:
    1. Generate embedding (OpenAI API)
    2. Prepare Qdrant point với payload
    3. Upsert to Qdrant collection
    4. Update indexed_at timestamp
    ↓
Return success
```

---

### 2.3 Chat with Campaign

**Endpoint:** `POST /api/v1/chat`  
**Purpose:** Hỏi đáp trong context của Campaign  
**Authentication:** JWT token (user authentication)

**Request:**
```json
{
  "campaign_id": "camp_001",
  "message": "VinFast bị đánh giá tiêu cực về gì?",
  "conversation_id": "conv_123",  // Optional: for multi-turn conversation
  "filters": {  // Optional: additional filters
    "sentiment": "NEGATIVE",
    "aspects": ["BATTERY", "PRICE"],
    "date_from": "2026-02-01T00:00:00Z",
    "date_to": "2026-02-15T23:59:59Z",
    "platforms": ["tiktok", "youtube"]
  },
  "options": {
    "max_results": 10,  // Number of context documents
    "temperature": 0.7,  // LLM temperature
    "include_citations": true,
    "language": "vi"
  }
}
```

**Response:**
```json
{
  "conversation_id": "conv_123",
  "message_id": "msg_456",
  "answer": "VinFast nhận nhiều đánh giá tiêu cực về:\n\n1. **PIN/NĂNG LƯỢNG** (45% negative mentions)\n   - Sụt pin nhanh: 'đi 100km hết gần 25% pin' [1]\n   - Dung lượng thấp hơn kỳ vọng [2]\n   - Thời gian sạc lâu [3]\n\n2. **GIÁ CẢ** (30% negative mentions)\n   - Giá cao so với đối thủ: 'Giá 1.2 tỷ thì hơi cao' [4]\n   - Không xứng đáng với giá tiền [5]\n\n3. **DỊCH VỤ** (15% negative mentions)\n   - Chậm trễ trong bảo hành [6]\n   - Thiếu phụ tùng thay thế [7]",
  "citations": [
    {
      "id": "analytics_001",
      "content": "Xe đẹp nhưng pin sụt nhanh, đi 100km hết gần 25% pin",
      "source": {
        "platform": "tiktok",
        "author": "nguyen_van_a_2024",
        "created_at": "2026-02-10T14:30:00Z",
        "url": "https://tiktok.com/@nguyen_van_a_2024/video/7234567890"
      },
      "relevance_score": 0.92,
      "sentiment": "NEGATIVE",
      "aspects": ["BATTERY"]
    },
    {
      "id": "analytics_002",
      "content": "Pin yếu hơn mình nghĩ, không đủ cho chuyến đi dài",
      "source": {
        "platform": "youtube",
        "author": "AutoReview VN",
        "created_at": "2026-02-08T10:00:00Z"
      },
      "relevance_score": 0.88,
      "sentiment": "NEGATIVE",
      "aspects": ["BATTERY"]
    }
  ],
  "metadata": {
    "total_documents_searched": 1500,
    "documents_used": 10,
    "processing_time_ms": 2340,
    "model_used": "gpt-4",
    "embedding_model": "text-embedding-3-small"
  },
  "suggestions": [
    "Có bao nhiêu người phàn nàn về pin?",
    "So sánh pin VinFast với BYD",
    "Xu hướng đánh giá về pin theo thời gian"
  ],
  "timestamp": "2026-02-15T10:50:00Z"
}
```

**Error Response:**
```json
{
  "error": {
    "code": "CAMPAIGN_NOT_FOUND",
    "message": "Campaign with ID 'camp_001' not found",
    "timestamp": "2026-02-15T10:50:00Z"
  }
}
```

**Flow:**
```
User sends message in Campaign UI
    ↓
POST /api/v1/chat
    ↓
Knowledge Service:
    1. Get campaign projects from Project Service
       GET /api/v1/campaigns/camp_001
       → project_ids: ["proj_vf8", "proj_byd"]
    
    2. Generate query embedding
       OpenAI API: text-embedding-3-small
    
    3. Search Qdrant with filters:
       - Vector similarity
       - project_id IN ["proj_vf8", "proj_byd"]
       - sentiment = "NEGATIVE" (if specified)
       - aspects IN ["BATTERY", "PRICE"] (if specified)
       - date range filter
    
    4. Get top 10 relevant documents
    
    5. Build context from documents
    
    6. Generate answer with LLM:
       OpenAI API: GPT-4
       Prompt: system + context + question
    
    7. Extract citations
    
    8. Save conversation history
    ↓
Return response
```

---

### 2.4 Get Conversation History

**Endpoint:** `GET /api/v1/conversations/{conversation_id}`  
**Purpose:** Lấy lịch sử chat  
**Authentication:** JWT token

**Response:**
```json
{
  "conversation_id": "conv_123",
  "campaign_id": "camp_001",
  "messages": [
    {
      "message_id": "msg_001",
      "role": "user",
      "content": "VinFast được đánh giá như thế nào?",
      "timestamp": "2026-02-15T10:45:00Z"
    },
    {
      "message_id": "msg_002",
      "role": "assistant",
      "content": "VinFast nhận được đánh giá trái chiều...",
      "citations": [...],
      "timestamp": "2026-02-15T10:45:05Z"
    },
    {
      "message_id": "msg_003",
      "role": "user",
      "content": "Cụ thể về pin thì sao?",
      "timestamp": "2026-02-15T10:46:00Z"
    },
    {
      "message_id": "msg_004",
      "role": "assistant",
      "content": "Về pin, VinFast nhận nhiều phản hồi tiêu cực...",
      "citations": [...],
      "timestamp": "2026-02-15T10:46:08Z"
    }
  ],
  "created_at": "2026-02-15T10:45:00Z",
  "updated_at": "2026-02-15T10:46:08Z"
}
```

---

### 2.5 Search Documents

**Endpoint:** `POST /api/v1/search`  
**Purpose:** Semantic search trong campaign (không generate answer)  
**Authentication:** JWT token

**Request:**
```json
{
  "campaign_id": "camp_001",
  "query": "pin sụt nhanh",
  "filters": {
    "project_ids": ["proj_vf8"],  // Optional: override campaign projects
    "sentiment": "NEGATIVE",
    "aspects": ["BATTERY"],
    "date_from": "2026-02-01T00:00:00Z",
    "date_to": "2026-02-15T23:59:59Z"
  },
  "limit": 20,
  "offset": 0
}
```

**Response:**
```json
{
  "total": 156,
  "results": [
    {
      "id": "analytics_001",
      "content": "Xe đẹp nhưng pin sụt nhanh, đi 100km hết gần 25% pin",
      "relevance_score": 0.94,
      "project_id": "proj_vf8",
      "source": {
        "platform": "tiktok",
        "author": "nguyen_van_a_2024",
        "created_at": "2026-02-10T14:30:00Z"
      },
      "sentiment": "NEGATIVE",
      "sentiment_score": -0.72,
      "aspects": [
        {
          "aspect": "BATTERY",
          "sentiment": "NEGATIVE",
          "score": -0.8
        }
      ],
      "keywords": ["pin", "sụt nhanh", "100km"],
      "engagement": {
        "views": 45000,
        "likes": 3200
      }
    }
  ],
  "aggregations": {
    "by_sentiment": {
      "POSITIVE": 12,
      "NEGATIVE": 120,
      "NEUTRAL": 24
    },
    "by_aspect": {
      "BATTERY": 156
    },
    "by_platform": {
      "tiktok": 89,
      "youtube": 45,
      "facebook": 22
    }
  },
  "processing_time_ms": 450
}
```

---

### 2.6 Generate Report

**Endpoint:** `POST /api/v1/reports/generate`  
**Purpose:** Tạo báo cáo tự động từ campaign data  
**Authentication:** JWT token

**Request:**
```json
{
  "campaign_id": "camp_001",
  "report_type": "COMPARISON",  // SUMMARY | COMPARISON | TREND | ASPECT_DEEP_DIVE
  "title": "Báo cáo so sánh VinFast vs BYD - Q1/2026",
  "filters": {
    "date_from": "2026-01-01T00:00:00Z",
    "date_to": "2026-03-31T23:59:59Z"
  },
  "options": {
    "format": "PDF",  // PDF | MARKDOWN | DOCX
    "language": "vi",
    "include_charts": true,
    "include_raw_data": false
  }
}
```

**Response (Async):**
```json
{
  "report_id": "report_789",
  "status": "PROCESSING",
  "estimated_time_seconds": 30,
  "message": "Đang tạo báo cáo, vui lòng đợi..."
}
```

**Check Status:**
```http
GET /api/v1/reports/{report_id}
```

**Response (Completed):**
```json
{
  "report_id": "report_789",
  "status": "COMPLETED",
  "title": "Báo cáo so sánh VinFast vs BYD - Q1/2026",
  "campaign_id": "camp_001",
  "file": {
    "url": "https://minio.smap.local/reports/report_789.pdf",
    "filename": "bao_cao_so_sanh_q1_2026.pdf",
    "size_bytes": 2456789,
    "format": "PDF"
  },
  "summary": {
    "total_mentions": 2500,
    "projects_analyzed": 2,
    "date_range": {
      "from": "2026-01-01T00:00:00Z",
      "to": "2026-03-31T23:59:59Z"
    }
  },
  "created_at": "2026-02-15T10:50:00Z",
  "completed_at": "2026-02-15T10:50:28Z"
}
```

**Report Structure (PDF):**
```
1. Executive Summary
   - Tổng quan so sánh
   - Key findings
   
2. Sentiment Analysis
   - Overall sentiment comparison
   - Sentiment trend over time
   
3. Aspect-by-Aspect Comparison
   - DESIGN: VinFast 85% vs BYD 72%
   - BATTERY: VinFast 45% vs BYD 68%
   - PRICE: VinFast 35% vs BYD 55%
   - SERVICE: VinFast 68% vs BYD 52%
   
4. Key Insights
   - VinFast strengths: Design, Service
   - VinFast weaknesses: Battery, Price
   - BYD strengths: Battery, Price
   - BYD weaknesses: Design
   
5. Recommendations
   - Focus areas for improvement
   - Marketing opportunities
   
6. Appendix
   - Top mentions
   - Data sources
   - Methodology
```

---

### 2.7 Get Smart Suggestions

**Endpoint:** `GET /api/v1/campaigns/{campaign_id}/suggestions`  
**Purpose:** Gợi ý câu hỏi thông minh dựa trên data  
**Authentication:** JWT token

**Response:**
```json
{
  "campaign_id": "camp_001",
  "suggestions": [
    {
      "category": "TRENDING",
      "question": "Tại sao VinFast bị đánh giá tiêu cực về pin?",
      "reason": "45% negative mentions về BATTERY aspect",
      "priority": "HIGH"
    },
    {
      "category": "COMPARISON",
      "question": "So sánh VinFast và BYD về giá cả",
      "reason": "Cả 2 brands đều có nhiều mentions về PRICE",
      "priority": "MEDIUM"
    },
    {
      "category": "TREND",
      "question": "Xu hướng đánh giá về VinFast tuần này",
      "reason": "Sentiment score tăng 15% so với tuần trước",
      "priority": "MEDIUM"
    },
    {
      "category": "INSIGHT",
      "question": "Điểm mạnh của VinFast so với đối thủ",
      "reason": "DESIGN aspect có 85% positive",
      "priority": "LOW"
    }
  ],
  "generated_at": "2026-02-15T10:50:00Z"
}
```

**Suggestion Logic:**
```
1. TRENDING: Aspects với nhiều negative mentions
2. COMPARISON: Aspects có sự chênh lệch lớn giữa brands
3. TREND: Sentiment changes > 10% trong 7 ngày
4. INSIGHT: Aspects với high positive score
```

---

### 2.8 Batch Index (Internal API)

**Endpoint:** `POST /internal/index/batch`  
**Purpose:** Index nhiều documents cùng lúc (bulk indexing)  
**Authentication:** Internal service token

**Request:**
```json
{
  "documents": [
    {
      "analytics_id": "analytics_001",
      "project_id": "proj_vf8",
      "content": "...",
      "sentiment": "NEGATIVE",
      "aspects": [...]
    },
    {
      "analytics_id": "analytics_002",
      "project_id": "proj_vf8",
      "content": "...",
      "sentiment": "POSITIVE",
      "aspects": [...]
    }
  ]
}
```

**Response:**
```json
{
  "success": true,
  "total": 100,
  "indexed": 98,
  "failed": 2,
  "errors": [
    {
      "analytics_id": "analytics_050",
      "error": "Content too short"
    },
    {
      "analytics_id": "analytics_075",
      "error": "Embedding generation failed"
    }
  ],
  "processing_time_ms": 15000
}
```

---


## 3. DATA MODELS

### 3.1 ChatRequest

```go
type ChatRequest struct {
    CampaignID     string                 `json:"campaign_id" binding:"required"`
    Message        string                 `json:"message" binding:"required"`
    ConversationID string                 `json:"conversation_id,omitempty"`
    Filters        *SearchFilters         `json:"filters,omitempty"`
    Options        *ChatOptions           `json:"options,omitempty"`
}

type SearchFilters struct {
    Sentiment   string    `json:"sentiment,omitempty"`    // POSITIVE | NEGATIVE | NEUTRAL | MIXED
    Aspects     []string  `json:"aspects,omitempty"`      // ["BATTERY", "PRICE"]
    DateFrom    time.Time `json:"date_from,omitempty"`
    DateTo      time.Time `json:"date_to,omitempty"`
    Platforms   []string  `json:"platforms,omitempty"`    // ["tiktok", "youtube"]
    ProjectIDs  []string  `json:"project_ids,omitempty"`  // Override campaign projects
}

type ChatOptions struct {
    MaxResults       int     `json:"max_results" default:"10"`
    Temperature      float64 `json:"temperature" default:"0.7"`
    IncludeCitations bool    `json:"include_citations" default:"true"`
    Language         string  `json:"language" default:"vi"`
}
```

### 3.2 ChatResponse

```go
type ChatResponse struct {
    ConversationID string       `json:"conversation_id"`
    MessageID      string       `json:"message_id"`
    Answer         string       `json:"answer"`
    Citations      []Citation   `json:"citations"`
    Metadata       *Metadata    `json:"metadata"`
    Suggestions    []string     `json:"suggestions"`
    Timestamp      time.Time    `json:"timestamp"`
}

type Citation struct {
    ID             string                 `json:"id"`
    Content        string                 `json:"content"`
    Source         *Source                `json:"source"`
    RelevanceScore float64                `json:"relevance_score"`
    Sentiment      string                 `json:"sentiment"`
    Aspects        []AspectInfo           `json:"aspects"`
}

type Source struct {
    Platform  string    `json:"platform"`
    Author    string    `json:"author"`
    CreatedAt time.Time `json:"created_at"`
    URL       string    `json:"url,omitempty"`
}

type Metadata struct {
    TotalDocumentsSearched int    `json:"total_documents_searched"`
    DocumentsUsed          int    `json:"documents_used"`
    ProcessingTimeMs       int    `json:"processing_time_ms"`
    ModelUsed              string `json:"model_used"`
    EmbeddingModel         string `json:"embedding_model"`
}
```

### 3.3 IndexRequest

```go
type IndexRequest struct {
    AnalyticsID           string                 `json:"analytics_id" binding:"required"`
    ProjectID             string                 `json:"project_id" binding:"required"`
    SourceID              string                 `json:"source_id" binding:"required"`
    Content               string                 `json:"content" binding:"required"`
    ContentCreatedAt      time.Time              `json:"content_created_at" binding:"required"`
    Platform              string                 `json:"platform" binding:"required"`
    OverallSentiment      string                 `json:"overall_sentiment"`
    OverallSentimentScore float64                `json:"overall_sentiment_score"`
    Aspects               []AspectAnalysis       `json:"aspects"`
    Keywords              []string               `json:"keywords"`
    Metadata              map[string]interface{} `json:"metadata"`
}

type AspectAnalysis struct {
    Aspect         string   `json:"aspect"`
    Sentiment      string   `json:"sentiment"`
    SentimentScore float64  `json:"sentiment_score"`
    Keywords       []string `json:"keywords"`
}
```

### 3.4 ReportRequest

```go
type ReportRequest struct {
    CampaignID string         `json:"campaign_id" binding:"required"`
    ReportType string         `json:"report_type" binding:"required"` // SUMMARY | COMPARISON | TREND | ASPECT_DEEP_DIVE
    Title      string         `json:"title" binding:"required"`
    Filters    *SearchFilters `json:"filters,omitempty"`
    Options    *ReportOptions `json:"options,omitempty"`
}

type ReportOptions struct {
    Format          string `json:"format" default:"PDF"`        // PDF | MARKDOWN | DOCX
    Language        string `json:"language" default:"vi"`
    IncludeCharts   bool   `json:"include_charts" default:"true"`
    IncludeRawData  bool   `json:"include_raw_data" default:"false"`
}
```

---

## 4. ERROR CODES

| Code | HTTP Status | Description | Solution |
|------|-------------|-------------|----------|
| `CAMPAIGN_NOT_FOUND` | 404 | Campaign không tồn tại | Kiểm tra campaign_id |
| `INVALID_REQUEST` | 400 | Request body không hợp lệ | Kiểm tra JSON format |
| `EMBEDDING_FAILED` | 500 | Không tạo được embedding | Retry hoặc check OpenAI API |
| `SEARCH_FAILED` | 500 | Qdrant search lỗi | Check Qdrant connection |
| `LLM_FAILED` | 500 | LLM generation lỗi | Retry hoặc check OpenAI API |
| `UNAUTHORIZED` | 401 | Token không hợp lệ | Login lại |
| `RATE_LIMIT_EXCEEDED` | 429 | Quá nhiều requests | Đợi và retry |
| `CONTENT_TOO_SHORT` | 400 | Content < 10 characters | Cung cấp content dài hơn |
| `NO_RESULTS_FOUND` | 404 | Không tìm thấy kết quả | Thử query khác |

---

## 5. AUTHENTICATION

### 5.1 JWT Token

Tất cả public endpoints yêu cầu JWT token trong header:

```http
Authorization: Bearer <jwt_token>
```

**Token Claims:**
```json
{
  "user_id": "user_123",
  "email": "analyst@vinfast.com",
  "role": "ANALYST",
  "exp": 1707577800
}
```

### 5.2 Internal Service Token

Internal endpoints yêu cầu service token:

```http
X-Service-Token: <service_secret>
```

---

## 6. RATE LIMITING

| Endpoint | Rate Limit | Window |
|----------|------------|--------|
| `/api/v1/chat` | 60 requests | 1 minute |
| `/api/v1/search` | 120 requests | 1 minute |
| `/api/v1/reports/generate` | 10 requests | 1 hour |
| `/internal/index` | 1000 requests | 1 minute |
| `/internal/index/batch` | 100 requests | 1 minute |

**Rate Limit Headers:**
```http
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1707577800
```

---

## 7. WEBHOOKS (Optional)

### 7.1 Report Completion Webhook

Khi report generation hoàn thành, Knowledge Service có thể gọi webhook:

**Webhook URL:** Configured in campaign settings

**Payload:**
```json
{
  "event": "REPORT_COMPLETED",
  "report_id": "report_789",
  "campaign_id": "camp_001",
  "status": "COMPLETED",
  "file_url": "https://minio.smap.local/reports/report_789.pdf",
  "timestamp": "2026-02-15T10:50:28Z"
}
```

---

## 8. PERFORMANCE METRICS

### 8.1 Expected Latencies

| Operation | P50 | P95 | P99 |
|-----------|-----|-----|-----|
| Index single document | 500ms | 1s | 2s |
| Search (no LLM) | 200ms | 500ms | 1s |
| Chat (with LLM) | 2s | 5s | 10s |
| Report generation | 20s | 60s | 120s |

### 8.2 Throughput

- **Indexing:** 100 documents/second (batch mode)
- **Search:** 50 queries/second
- **Chat:** 10 conversations/second

---

## 9. MONITORING & OBSERVABILITY

### 9.1 Prometheus Metrics

```
# Request metrics
knowledge_requests_total{endpoint, method, status}
knowledge_request_duration_seconds{endpoint}

# Embedding metrics
knowledge_embeddings_generated_total
knowledge_embedding_duration_seconds

# Search metrics
knowledge_searches_total
knowledge_search_duration_seconds
knowledge_search_results_count

# LLM metrics
knowledge_llm_calls_total{model}
knowledge_llm_duration_seconds{model}
knowledge_llm_tokens_used{model, type}  # type: prompt | completion

# Error metrics
knowledge_errors_total{type, endpoint}
```

### 9.2 Health Check Details

```http
GET /health/detailed
```

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime_seconds": 86400,
  "dependencies": {
    "qdrant": {
      "status": "connected",
      "latency_ms": 5,
      "collections": ["smap_analytics"],
      "total_vectors": 150000
    },
    "openai": {
      "status": "available",
      "models": ["text-embedding-3-small", "gpt-4"]
    },
    "postgres": {
      "status": "connected",
      "latency_ms": 2
    },
    "redis": {
      "status": "connected",
      "latency_ms": 1
    }
  },
  "metrics": {
    "total_requests_today": 5420,
    "total_embeddings_generated": 1200,
    "total_searches": 3800,
    "total_chats": 420,
    "cache_hit_rate": 0.65
  }
}
```

---

## 10. CACHING STRATEGY

### 10.1 Embedding Cache (Redis)

```
Key: embedding:{content_hash}
Value: [0.123, -0.456, ...]
TTL: 7 days
```

**Benefits:**
- Giảm OpenAI API calls
- Tăng tốc indexing
- Tiết kiệm chi phí

### 10.2 Search Results Cache (Redis)

```
Key: search:{campaign_id}:{query_hash}:{filters_hash}
Value: {results: [...], timestamp: ...}
TTL: 5 minutes
```

**Invalidation:**
- Khi có data mới được index
- Khi campaign được update

---

## 11. EXAMPLE WORKFLOWS

### 11.1 Complete Chat Flow

```bash
# 1. User sends message
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "campaign_id": "camp_001",
    "message": "VinFast bị đánh giá tiêu cực về gì?"
  }'

# 2. Response with answer and citations
{
  "answer": "VinFast nhận nhiều đánh giá tiêu cực về PIN (45%), GIÁ (30%)...",
  "citations": [...]
}

# 3. Follow-up question
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "campaign_id": "camp_001",
    "conversation_id": "conv_123",
    "message": "Cụ thể về pin thì sao?"
  }'
```

### 11.2 Report Generation Flow

```bash
# 1. Request report generation
curl -X POST http://localhost:8080/api/v1/reports/generate \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "campaign_id": "camp_001",
    "report_type": "COMPARISON",
    "title": "Báo cáo so sánh Q1/2026"
  }'

# Response: {"report_id": "report_789", "status": "PROCESSING"}

# 2. Poll for status
curl http://localhost:8080/api/v1/reports/report_789 \
  -H "Authorization: Bearer $JWT_TOKEN"

# 3. Download when completed
curl http://localhost:8080/api/v1/reports/report_789/download \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -o report.pdf
```

---

## 12. DEPLOYMENT CONFIGURATION

### 12.1 Environment Variables

```bash
# Service
PORT=8080
LOG_LEVEL=info

# OpenAI
OPENAI_API_KEY=sk-...
OPENAI_EMBEDDING_MODEL=text-embedding-3-small
OPENAI_LLM_MODEL=gpt-4
OPENAI_MAX_RETRIES=3

# Qdrant
QDRANT_URL=http://qdrant:6333
QDRANT_COLLECTION=smap_analytics
QDRANT_API_KEY=...

# PostgreSQL
POSTGRES_URL=postgresql://user:pass@postgres:5432/smap
POSTGRES_MAX_CONNECTIONS=20

# Redis
REDIS_URL=redis://redis:6379/0
REDIS_CACHE_TTL=300

# Project Service
PROJECT_SERVICE_URL=http://project-service:8080

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=60
```

### 12.2 Docker Compose

```yaml
version: '3.8'

services:
  knowledge-service:
    image: smap/knowledge-service:1.0.0
    ports:
      - "8080:8080"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - QDRANT_URL=http://qdrant:6333
      - POSTGRES_URL=postgresql://postgres:5432/smap
      - REDIS_URL=redis://redis:6379/0
    depends_on:
      - qdrant
      - postgres
      - redis
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

---

## 13. TESTING

### 13.1 Unit Tests

```bash
# Run unit tests
go test ./internal/... -v

# With coverage
go test ./internal/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### 13.2 Integration Tests

```bash
# Start test dependencies
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test ./tests/integration/... -v

# Cleanup
docker-compose -f docker-compose.test.yml down
```

### 13.3 Load Tests

```bash
# Install k6
brew install k6

# Run load test
k6 run tests/load/chat_test.js

# Expected results:
# - P95 latency < 5s
# - Error rate < 1%
# - Throughput > 10 req/s
```

---

## 14. TROUBLESHOOTING

### 14.1 Common Issues

**Issue 1: Slow chat responses**
```
Symptoms: Chat latency > 10s
Causes:
  - OpenAI API slow
  - Qdrant search slow
  - Too many context documents

Solutions:
  - Reduce max_results
  - Enable caching
  - Use faster embedding model
  - Optimize Qdrant indexes
```

**Issue 2: Low relevance scores**
```
Symptoms: Citations không liên quan
Causes:
  - Query embedding không tốt
  - Filters quá strict
  - Data quality thấp

Solutions:
  - Improve query preprocessing
  - Relax filters
  - Re-index with better embeddings
```

**Issue 3: High OpenAI costs**
```
Symptoms: API bill cao
Causes:
  - Không cache embeddings
  - Generate quá nhiều tokens
  - Sử dụng GPT-4 cho mọi query

Solutions:
  - Enable Redis caching
  - Reduce max_tokens
  - Use GPT-3.5-turbo cho simple queries
```

---

## 15. ROADMAP

### Phase 1 (Current)
- ✅ Basic RAG pipeline
- ✅ Campaign-scoped search
- ✅ Chat with citations
- ✅ Report generation

### Phase 2 (Next 3 months)
- [ ] Multi-turn conversation với context
- [ ] Smart suggestions based on visual data
- [ ] Artifact editing (inline + Google Docs)
- [ ] Advanced filters (engagement, influence)

### Phase 3 (6 months)
- [ ] Multi-language support
- [ ] Custom LLM fine-tuning
- [ ] Real-time streaming responses
- [ ] Voice input/output

---

**Document Version:** 1.0.0  
**Last Updated:** 2026-02-15  
**Maintainer:** SMAP Team
