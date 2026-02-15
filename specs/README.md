# Input/Output Documentation - Knowledge Service

Thư mục này chứa các tài liệu chi tiết về data models, API specifications, và examples cho Knowledge Service (RAG).

---

## 📚 Danh sách Documents

### 1. **KNOWLEDGE_SERVICE_API_SPEC.md**
**Mục đích:** API specification đầy đủ cho Knowledge Service

**Nội dung:**
- ✅ Tất cả endpoints (public + internal)
- ✅ Request/Response models
- ✅ Authentication & authorization
- ✅ Error codes
- ✅ Rate limiting
- ✅ Performance metrics
- ✅ Monitoring & observability
- ✅ Deployment configuration
- ✅ Testing strategies
- ✅ Troubleshooting guide

**Sử dụng cho:**
- Backend developers implement API
- Frontend developers integrate với API
- QA team viết test cases
- DevOps team setup monitoring

---

### 2. **KNOWLEDGE_SERVICE_DEPENDENCIES.md** ⭐ QUAN TRỌNG
**Mục đích:** Chi tiết TẤT CẢ dependencies của Knowledge Service

**Nội dung:**
- ✅ 6 dependencies BẮT BUỘC (Qdrant, Voyage AI, Gemini, PostgreSQL, Redis, Project Service, MinIO)
- ✅ 3 dependencies OPTIONAL (Kafka, Prometheus, Jaeger)
- ✅ Vai trò từng dependency
- ✅ Code examples kết nối
- ✅ Config examples
- ✅ Docker Compose setup
- ✅ Health check implementation

**Sử dụng cho:**
- DevOps team setup infrastructure
- Backend developers understand architecture
- Troubleshooting dependency issues

---

### 3. **MODEL_PROVIDERS_COMPARISON.md** ⭐ MỚI
**Mục đích:** So sánh và lý do chọn Voyage AI + Gemini thay vì OpenAI

**Nội dung:**
- ✅ So sánh chi tiết: Voyage AI vs OpenAI (embedding)
- ✅ So sánh chi tiết: Gemini vs GPT-4 (LLM)
- ✅ Cost analysis: Tiết kiệm 80-99%!
- ✅ Performance benchmarks
- ✅ Implementation guide
- ✅ Migration checklist

**Key Highlights:**
- Voyage AI: SOTA retrieval, $0.10/1M tokens
- Gemini 1.5 Pro: FREE tier, 2M context
- Total savings: $6,000-7,500/month!

---

### 4. **analytics_post_example.json**
**Mục đích:** Ví dụ JSON data thực tế của analytics output

**Nội dung:**
- ✅ Complete analytics record với tất cả fields
- ✅ Comments giải thích từng field
- ✅ Real-world data example (VinFast review)
- ✅ Aspect-based sentiment analysis
- ✅ Risk assessment
- ✅ Engagement metrics

**Sử dụng cho:**
- Hiểu structure của analytics data
- Test data cho development
- Documentation reference

---

### 3. **ANALYTICS_SCHEMA_EXPLAINED.md**
**Mục đích:** Giải thích chi tiết schema analytics.post_analytics

**Nội dung:**
- ✅ Ý nghĩa từng field group
- ✅ Cách sử dụng trong RAG
- ✅ SQL query examples
- ✅ RAG query examples
- ✅ PostgreSQL schema definition
- ✅ Indexing strategies

**Sử dụng cho:**
- Backend developers implement analytics pipeline
- Data analysts query data
- RAG developers build filters
- Database administrators optimize schema

---

## 🔄 Data Flow Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    COMPLETE DATA FLOW                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. RAW DATA                                                    │
│     Excel/CSV/TikTok/YouTube/...                                │
│                                                                 │
│  2. UAP (Unified Analytics Payload)                             │
│     {                                                           │
│       "content": "...",                                         │
│       "content_created_at": "...",                              │
│       "platform": "tiktok"                                      │
│     }                                                           │
│                                                                 │
│  3. ANALYTICS PIPELINE                                          │
│     Sentiment + Aspect + Keyword extraction                     │
│                                                                 │
│  4. ANALYTICS OUTPUT (analytics.post_analytics)                 │
│     {                                                           │
│       "overall_sentiment": "NEGATIVE",                          │
│       "aspects": [                                              │
│         {"aspect": "BATTERY", "sentiment": "NEGATIVE"}          │
│       ]                                                         │
│     }                                                           │
│     ↓                                                           │
│     📄 See: analytics_post_example.json                         │
│     📖 See: ANALYTICS_SCHEMA_EXPLAINED.md                       │
│                                                                 │
│  5. KNOWLEDGE SERVICE INDEXING                                  │
│     POST /internal/index                                        │
│     ↓                                                           │
│     📄 See: KNOWLEDGE_SERVICE_API_SPEC.md (Section 2.2)         │
│                                                                 │
│  6. QDRANT VECTOR DB                                            │
│     {                                                           │
│       "vector": [0.123, -0.456, ...],                           │
│       "payload": {                                              │
│         "project_id": "proj_vf8",                               │
│         "sentiment": "NEGATIVE",                                │
│         "aspects": [...]                                        │
│       }                                                         │
│     }                                                           │
│                                                                 │
│  7. RAG QUERY                                                   │
│     POST /api/v1/chat                                           │
│     {                                                           │
│       "campaign_id": "camp_001",                                │
│       "message": "VinFast bị đánh giá tiêu cực về gì?"         │
│     }                                                           │
│     ↓                                                           │
│     📄 See: KNOWLEDGE_SERVICE_API_SPEC.md (Section 2.3)         │
│                                                                 │
│  8. RAG RESPONSE                                                │
│     {                                                           │
│       "answer": "VinFast nhận đánh giá tiêu cực về PIN...",    │
│       "citations": [...]                                        │
│     }                                                           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🎯 Use Cases & Examples

### Use Case 1: Simple Sentiment Query

**User Question:** "VinFast được đánh giá tích cực về gì?"

**API Call:**
```bash
POST /api/v1/chat
{
  "campaign_id": "camp_001",
  "message": "VinFast được đánh giá tích cực về gì?",
  "filters": {
    "sentiment": "POSITIVE"
  }
}
```

**RAG Process:**
1. Embed query → vector
2. Search Qdrant với filter: `sentiment = "POSITIVE"`
3. Get top 10 positive mentions
4. Generate answer từ context

**Response:**
```
"VinFast được đánh giá tích cực về:
1. THIẾT KẾ (85% positive) - Đẹp, sang trọng
2. DỊCH VỤ (68% positive) - Nhân viên nhiệt tình
3. CÔNG NGHỆ (72% positive) - Tính năng hiện đại"
```

---

### Use Case 2: Aspect-Specific Query

**User Question:** "Tại sao VinFast bị chê về pin?"

**API Call:**
```bash
POST /api/v1/chat
{
  "campaign_id": "camp_001",
  "message": "Tại sao VinFast bị chê về pin?",
  "filters": {
    "aspects": ["BATTERY"],
    "sentiment": "NEGATIVE"
  }
}
```

**RAG Process:**
1. Embed query → vector
2. Search Qdrant với filters:
   - `aspects[].aspect = "BATTERY"`
   - `aspects[].sentiment = "NEGATIVE"`
3. Extract mentions từ aspects
4. Generate detailed answer

**Response:**
```
"VinFast bị chê về pin vì:
1. Sụt pin nhanh - 'đi 100km hết 25% pin' [1]
2. Dung lượng thấp - 'pin yếu hơn kỳ vọng' [2]
3. Sạc chậm - 'sạc đầy mất 8 tiếng' [3]

Có 320 mentions tiêu cực về pin với avg score -0.72"
```

---

### Use Case 3: Comparison Query (Campaign-Scoped)

**User Question:** "So sánh VinFast và BYD về giá"

**Campaign Setup:**
```
Campaign: "So sánh Xe điện Q1/2026"
Projects: ["proj_vf8", "proj_byd"]
```

**API Call:**
```bash
POST /api/v1/chat
{
  "campaign_id": "camp_001",
  "message": "So sánh VinFast và BYD về giá",
  "filters": {
    "aspects": ["PRICE"]
  }
}
```

**RAG Process:**
1. Get campaign projects: ["proj_vf8", "proj_byd"]
2. Search Qdrant với filters:
   - `project_id IN ["proj_vf8", "proj_byd"]`
   - `aspects[].aspect = "PRICE"`
3. Group results by project_id
4. Aggregate sentiment scores
5. Generate comparison

**Response:**
```
"So sánh giá:

VinFast VF8:
- Giá: 1.2 tỷ
- Sentiment: 65% negative
- Phản hồi: 'Đắt so với đối thủ', 'Giá cao'

BYD Seal:
- Giá: 900 triệu
- Sentiment: 45% negative
- Phản hồi: 'Giá hợp lý', 'Rẻ hơn VinFast'

Kết luận: BYD được đánh giá tốt hơn về giá"
```

---

## 🔧 Quick Start Guide

### 1. Understand Data Structure

```bash
# Read analytics schema
cat ANALYTICS_SCHEMA_EXPLAINED.md

# See example data
cat analytics_post_example.json
```

### 2. Review API Specification

```bash
# Read API spec
cat KNOWLEDGE_SERVICE_API_SPEC.md

# Focus on these sections:
# - Section 2: API Endpoints
# - Section 3: Data Models
# - Section 11: Example Workflows
```

### 3. Test API Locally

```bash
# Start Knowledge Service
docker-compose up knowledge-service

# Test health check
curl http://localhost:8080/health

# Test chat
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d @test_chat_request.json
```

### 4. Integrate with Frontend

```typescript
// Example: React integration
import { KnowledgeServiceClient } from '@/lib/knowledge-service';

const client = new KnowledgeServiceClient({
  baseURL: 'http://localhost:8080',
  token: userToken
});

// Send chat message
const response = await client.chat({
  campaign_id: 'camp_001',
  message: 'VinFast bị đánh giá tiêu cực về gì?',
  filters: {
    sentiment: 'NEGATIVE'
  }
});

console.log(response.answer);
console.log(response.citations);
```

---

## 📊 Data Models Summary

### Analytics Output (Input to Knowledge Service)

```typescript
interface AnalyticsPost {
  // Identity
  id: string;
  project_id: string;
  source_id: string;
  
  // UAP Core
  content: string;
  content_created_at: Date;
  platform: string;
  
  // Sentiment
  overall_sentiment: 'POSITIVE' | 'NEGATIVE' | 'NEUTRAL' | 'MIXED';
  overall_sentiment_score: number;  // -1.0 to 1.0
  
  // ABSA
  aspects: AspectAnalysis[];
  
  // Keywords
  keywords: string[];
  
  // Metadata
  uap_metadata: Record<string, any>;
}

interface AspectAnalysis {
  aspect: string;              // "DESIGN", "BATTERY", "PRICE"...
  sentiment: string;           // "POSITIVE" | "NEGATIVE" | "NEUTRAL"
  sentiment_score: number;     // -1.0 to 1.0
  keywords: string[];
  mentions: Mention[];
}
```

### RAG Request/Response

```typescript
interface ChatRequest {
  campaign_id: string;
  message: string;
  conversation_id?: string;
  filters?: SearchFilters;
  options?: ChatOptions;
}

interface ChatResponse {
  conversation_id: string;
  message_id: string;
  answer: string;
  citations: Citation[];
  suggestions: string[];
  metadata: ResponseMetadata;
}

interface Citation {
  id: string;
  content: string;
  source: Source;
  relevance_score: number;
  sentiment: string;
  aspects: AspectInfo[];
}
```

---

## 🚀 Performance Benchmarks

### Expected Latencies

| Operation | P50 | P95 | P99 |
|-----------|-----|-----|-----|
| Index document | 500ms | 1s | 2s |
| Search (no LLM) | 200ms | 500ms | 1s |
| Chat (with LLM) | 2s | 5s | 10s |
| Report generation | 20s | 60s | 120s |

### Throughput

- **Indexing:** 100 documents/second (batch mode)
- **Search:** 50 queries/second
- **Chat:** 10 conversations/second

---

## 🔍 Troubleshooting

### Common Issues

**Issue 1: Slow responses**
- Check OpenAI API latency
- Enable Redis caching
- Reduce max_results

**Issue 2: Low relevance**
- Improve query preprocessing
- Tune Qdrant search parameters
- Re-index with better embeddings

**Issue 3: High costs**
- Cache embeddings in Redis
- Use GPT-3.5-turbo for simple queries
- Reduce max_tokens

---

## 📞 Support

**Questions?** Contact SMAP Team

**Issues?** Check troubleshooting section in API spec

**Updates?** Watch this directory for new documents

---

**Last Updated:** 2026-02-15  
**Version:** 1.0.0
