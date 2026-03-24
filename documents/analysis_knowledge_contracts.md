# Analysis → Knowledge Service: Payload Contracts

> **Version:** 1.0  
> **Date:** 2026-03-24  
> **Status:** Approved  
> **Consumers:** knowledge-srv  
> **Producers:** analysis-srv (batch pipeline)

---

## Tổng Quan

Ngoài luồng micro-level đã có (`analytics.batch.completed`), hệ thống bổ sung **2 Kafka topic mới** để gửi tri thức macro-level (tổng hợp) từ Analysis sang Knowledge:

| # | Topic | Grain | Mục đích |
| :-- | :-- | :-- | :-- |
| Existing | `analytics.batch.completed` | Per-batch | Thông báo batch hoàn thành, knowledge-srv tải file JSONL từ MinIO rồi index |
| Existing | `smap.analytics.output` | Per-post (Batch) | Stream dữ liệu enriched ngay khi xử lý xong từng post (JSON array) |
| **New 1** | `analytics.insights.published` | Per-insight-card | Gửi thẻ insight narrative (đã "tiêu hóa") để embed làm Synthesized Knowledge |
| **New 2** | `analytics.report.digest` | Per-run | Gửi bản tóm tắt curated của toàn bộ đợt phân tích (top entities, topics, issues) |

### Luồng dữ liệu tổng thể

```text
                        analysis-srv (Python)
                               │
            ┌──────────────────┼──────────────────────┐
            │                  │                      │
   [existing]              [NEW 1]                [NEW 2]
analytics.batch.     analytics.insights.    analytics.report.
  completed             published              digest
            │                  │                      │
            └──────────────────┼──────────────────────┘
                               │
                        knowledge-srv (Go)
                               │
                    ┌──────────┼──────────┐
                    │          │          │
                 Qdrant     Qdrant     Qdrant
               (per-post)  (macro_    (macro_
                collection  insights)  insights)
```

---

## Contract 0 (Existing): `analytics.batch.completed`

### Kafka Configuration (Contract 0)

| Field | Value |
| :--- | :--- |
| **Topic** | `analytics.batch.completed` |
| **Consumer Group** | `knowledge-indexing-batch` |
| **Key** | `{project_id}` |
| **Format** | JSON (single object) |
| **Compression** | gzip |
| **Publish timing** | 1 message khi analysis-srv xử lý xong toàn bộ batch |

### Payload Schema (Contract 0)

```jsonc
{
  "batch_id": "batch_20260323_001",        // string, bắt buộc — ID của batch đã xử lý
  "project_id": "proj_cleanser_01",        // string, bắt buộc — project sở hữu data
  "campaign_id": "camp_q1_2026",           // string, optional — campaign liên kết
  "file_url": "minio://enriched/proj_cleanser_01/batch_20260323_001.jsonl",
                                           // string, bắt buộc — URL file JSONL trên MinIO
                                           //   chứa array các InsightMessage (enriched per-post)
  "record_count": 2000,                    // int, bắt buộc — số record trong file
  "completed_at": "2026-03-23T19:00:49Z"   // string ISO8601, bắt buộc — thời điểm hoàn thành
}
```

### File JSONL trên MinIO (mỗi dòng là 1 InsightMessage)

Mỗi dòng trong file JSONL tại `file_url` có cấu trúc của `InsightMessage` — đây là payload **per-post** (micro-level):

```jsonc
{
  "message_version": "1.0",
  "event_id": "uuid-per-post",

  "project": {
    "project_id": "proj_cleanser_01",
    "entity_type": "product",
    "entity_name": "Facial Cleanser",
    "brand": "Multi-brand",
    "campaign_id": "camp_q1_2026"          // optional
  },

  "identity": {
    "source_type": "TIKTOK",
    "source_id": "src_tiktok_01",
    "doc_id": "tt_p_fc_0001",
    "doc_type": "post",                    // "post" | "comment" | "reply"
    "url": "https://tiktok.com/...",       // optional
    "language": "vi",                      // optional
    "published_at": "2026-02-07T09:55:00Z",
    "ingested_at": "2026-02-07T10:00:00Z",
    "author": {
      "author_id": "author_0001",
      "display_name": "Nguyễn A",
      "author_type": "user"               // "user" | "brand" | "kol"
    }
  },

  "content": {
    "text": "Xe đi êm nhưng pin sụt nhanh...",       // Nội dung gốc
    "clean_text": "Xe đi êm nhưng pin sụt nhanh...", // Nội dung đã clean
    "summary": "Người dùng khen xe êm nhưng chê..."   // Tóm tắt AI
  },

  "nlp": {
    "sentiment": {
      "label": "NEGATIVE",                 // POSITIVE | NEGATIVE | NEUTRAL | MIXED
      "score": -0.45,                      // float, -1.0 to 1.0
      "confidence": "HIGH",               // LOW | MEDIUM | HIGH
      "explanation": "Nội dung chứa..."    // optional
    },
    "aspects": [
      {
        "aspect": "BATTERY",               // string, aspect key
        "polarity": "NEGATIVE",            // POSITIVE | NEGATIVE | NEUTRAL
        "confidence": 0.74,                // float, 0.0–1.0
        "evidence": "pin sụt nhanh"        // string, đoạn trích dẫn
      }
    ],
    "entities": [
      {
        "type": "PRODUCT",                 // PRODUCT | BRAND | PERSON | LOCATION
        "value": "VF8",
        "confidence": 0.92
      }
    ]
  },

  "business": {
    "impact": {
      "engagement": {
        "like_count": 120,
        "comment_count": 34,
        "share_count": 5,
        "view_count": 1111
      },
      "impact_score": 68.5,               // float, 0–100
      "priority": "HIGH"                   // LOW | MEDIUM | HIGH | CRITICAL
    },
    "alerts": [
      {
        "alert_type": "NEGATIVE_BRAND_SIGNAL",
        "severity": "MEDIUM",              // LOW | MEDIUM | HIGH | CRITICAL
        "reason": "Bài viết có sentiment NEGATIVE...",
        "suggested_action": "Theo dõi thêm..."
      }
    ]
  },

  "rag": {
    "index": {
      "should_index": true,                // bool — knowledge-srv dựa vào cờ này
      "quality_gate": {
        "min_length_ok": true,
        "has_aspect": true,
        "not_spam": true
      }
    },
    "citation": {
      "source": "TikTok",
      "title": "TikTok Post",
      "snippet": "Xe đi êm nhưng pin sụt nhanh...",
      "url": "https://tiktok.com/...",
      "published_at": "2026-02-07T09:55:00Z"
    },
    "vector_ref": {
      "provider": "qdrant",
      "collection": "proj_cleanser_01",    // string, tên Qdrant collection
      "point_id": ""                       // string, được knowledge-srv gán sau khi index
    }
  },

  "provenance": {
    "raw_ref": "minio://raw/proj_cleanser_01/tiktok/2026-02-07/batch_001.jsonl",
    "pipeline": [
      { "step": "normalize_uap", "model": "", "at": "2026-02-07T10:00:30Z" },
      { "step": "sentiment_analysis", "model": "phobert-sentiment-v1", "at": "2026-02-07T10:00:32Z" },
      { "step": "aspect_extraction", "model": "phobert-aspect-v1", "at": "2026-02-07T10:00:34Z" },
      { "step": "embedding", "model": "text-embedding-3-large", "at": "2026-02-07T10:00:36Z" }
    ]
  }
}
```

### Processing Strategy (knowledge-srv hiện tại)

1. Consume `BatchCompletedMessage` từ Kafka
2. Tải file JSONL từ MinIO via `file_url`
3. Với mỗi dòng (InsightMessage): kiểm tra `rag.index.should_index`
4. Nếu `true`: embed `content.clean_text` bằng Voyage AI → upsert vào Qdrant collection `rag.vector_ref.collection`
5. Lưu metadata (sentiment, aspects, entities, citation) vào Qdrant point payload

### Code References

- **Go struct:** [type.go](file:///Users/tailung/Workspaces/smap-hcmut/knowledge-srv/internal/indexing/delivery/kafka/type.go) — `BatchCompletedMessage`
- **Go consumer:** [consumer.go](file:///Users/tailung/Workspaces/smap-hcmut/knowledge-srv/internal/indexing/delivery/kafka/consumer/consumer.go) — `ConsumeBatchCompleted()`
- **Go presenter:** [presenters.go](file:///Users/tailung/Workspaces/smap-hcmut/knowledge-srv/internal/indexing/delivery/kafka/consumer/presenters.go) — `toIndexInput()`
- **Python model:** [insight_message.py](file:///Users/tailung/Workspaces/smap-hcmut/analysis-srv/internal/model/insight_message.py) — `InsightMessage` dataclass

---

## Contract 0.1 (Existing): `smap.analytics.output`

### Kafka Configuration (Contract 0.1)

| Field | Value |
| :-- | :-- |
| **Topic** | `smap.analytics.output` |
| **Consumer Group** | N/A (Tương lai Knowledge Service sẽ consume) |
| **Key** | `{project_id}` |
| **Format** | JSON Array of `InsightMessage` |
| **Compression** | gzip |
| **Publish timing** | Stream liên tục ngay khi có bài viết được phân tích xong |

### Payload Schema (Contract 0.1)

Payload của topic này là một **JSON Array** chứa các object `InsightMessage`. Cấu trúc `InsightMessage` hoàn toàn giống với nội dung trong file JSONL tại MinIO (xem Contract 0 phía trên).

```json
[
  {
    "message_version": "1.0",
    "event_id": "post-001",
    "project": { ... },
    "identity": { ... },
    "content": { ... },
    "nlp": { ... },
    "business": { ... },
    "rag": { ... },
    "provenance": { ... }
  },
  {
    "message_version": "1.0",
    "event_id": "post-002",
    ...
  }
]
```

### So sánh với Contract 0

- **Contract 0 (`batch.completed`)**: Dành cho xử lý Offline/Batch. Toàn bộ dữ liệu được gom vào file lớn đưa lên MinIO. Knowledge Service rảnh lúc nào thì tải về index lúc đó.
- **Contract 0.1 (`smap.analytics.output`)**: Dành cho xử lý Real-time/Stream. Dữ liệu chảy liên tục. Thích hợp cho các tính năng Notification hoặc Live Search.

---

## Contract 1 (New): `analytics.insights.published`

### Kafka Configuration (Contract 1)

| Field | Value |
| :--- | :--- |
| **Topic** | `analytics.insights.published` |
| **Consumer Group** | `knowledge-indexing-insights` |
| **Key** | `{project_id}` |
| **Format** | JSON (single object, NOT array) |
| **Compression** | gzip |
| **Publish timing** | Sau khi batch pipeline hoàn tất, mỗi insight card = 1 message |

### Payload Schema (Contract 1)

```jsonc
{
  // ── Envelope ──
  "contract_version": "1.0",           // string, bắt buộc

  // ── Scope ──
  "project_id": "proj_cleanser_01",    // string, bắt buộc
  "campaign_id": "camp_q1_2026",       // string, optional
  "run_id": "run-20260323T165146Z",    // string, bắt buộc — liên kết về run_manifest

  // ── Insight Core ──
  "insight_type": "share_of_voice_shift",  // string, bắt buộc
  // Enum values:
  //   - "share_of_voice_shift"  : Biến động SOV của entity
  //   - "trending_topic"        : Topic đang tăng volume
  //   - "conversation_hotspot"  : Topic có buzz score cao
  //   - "emerging_topic"        : Topic mới xuất hiện
  //   - "issue_warning"         : Issue có áp lực cao
  //   - "controversy_alert"     : Thread có tranh cãi
  //   - "creator_concentration" : Tập trung tác giả

  "title": "Share of voice shift detected for Cetaphil",  // string, bắt buộc
  // Tiêu đề ngắn, human-readable

  "summary": "Cetaphil lost 22 mention(s) in the later half of the window.",  // string, bắt buộc
  // Nội dung narrative — đây là phần CHÍNH để embed vào vector DB

  "confidence": 0.5612,  // float, 0.0–1.0

  // ── Time Scope ──
  "analysis_window_start": "2026-02-01T10:17:00Z",  // string ISO8601, bắt buộc
  "analysis_window_end": "2026-02-10T21:24:00Z",    // string ISO8601, bắt buộc

  // ── Supporting Data ──
  "supporting_metrics": {               // object, flexible key-value
    "mention_share": 0.117,
    "delta_mention_share": -0.0112,
    "delta_mention_count": -22
  },

  "evidence_references": [              // string[], IDs bài viết gốc làm bằng chứng
    "tt_p_fc_0001",
    "tt_c_fc_0001_02"
  ],

  "source_reports": [                   // string[], tên report gốc tạo ra insight
    "sov_report",
    "buzz_report"
  ],

  "filters_used": {                     // object, bộ lọc áp dụng
    "canonical_entity_id": "brand.cetaphil"
  },

  // ── RAG Hints ──
  "rag_collection": "macro_insights",   // string, tên Qdrant collection đích
  "rag_document_type": "insight_card",  // string, phân loại document trong collection
  "should_index": true                  // bool, cờ quyết định có index hay không
}
```

### Embedding Strategy (cho knowledge-srv)

Khi consume message này, knowledge-srv nên:

1. **Concatenate** `title` + `summary` thành văn bản để embed
2. **Metadata payload** (lưu vào Qdrant point metadata, không embed):
   - `project_id`, `run_id`, `insight_type`, `confidence`
   - `analysis_window_start`, `analysis_window_end`
   - `source_reports`, `evidence_references`
3. **Point ID format:** `insight:{run_id}:{insight_type}:{hash(title)}`
4. **TTL/Overwrite:** Khi có run mới, insight cũ cùng `project_id` + `insight_type` có thể được thay thế (upsert by metadata filter)

---

## Contract 2 (New): `analytics.report.digest`

### Kafka Configuration (Contract 2)

| Field | Value |
| :--- | :--- |
| **Topic** | `analytics.report.digest` |
| **Consumer Group** | `knowledge-indexing-digest` |
| **Key** | `{project_id}` |
| **Format** | JSON (single object) |
| **Compression** | gzip |
| **Publish timing** | 1 message per run, sau khi toàn bộ BI reports được sinh |

### Payload Schema (Contract 2)

```jsonc
{
  // ── Envelope ──
  "contract_version": "1.0",
  
  // ── Scope ──
  "project_id": "proj_cleanser_01",
  "campaign_id": "camp_q1_2026",       // optional
  "run_id": "run-20260323T165146Z",
  
  // ── Time Scope ──
  "analysis_window_start": "2026-02-01T10:17:00Z",
  "analysis_window_end": "2026-02-10T21:24:00Z",
  
  // ── Domain Context ──
  "domain_overlay": "domain-facial-cleanser-vn",  // string, từ run_manifest.ontology_runtime
  "platform": "tiktok",                           // string, platform chính
  "total_mentions": 2000,                          // int, tổng số mention trong slice

  // ── Top Entities (max 10, sorted by mention_share DESC) ──
  "top_entities": [
    {
      "canonical_entity_id": "brand.cetaphil",    // string
      "entity_name": "Cetaphil",                  // string, display name
      "entity_type": "brand",                     // "brand" | "product"
      "mention_count": 234,                       // int
      "mention_share": 0.117,                     // float, 0.0–1.0
      "delta_mention_count": -22                  // int, so với prior half
    },
    {
      "canonical_entity_id": "brand.cerave",
      "entity_name": "CeraVe",
      "entity_type": "brand",
      "mention_count": 233,
      "mention_share": 0.1165,
      "delta_mention_count": 37
    }
    // ... max 10 items
  ],
  
  // ── Top Topics (max 10, sorted by mention_share DESC) ──
  "top_topics": [
    {
      "topic_key": "cleanser_recommendation_by_skin_type",     // string
      "topic_label": "Cleanser Recommendation by Skin Type",   // string, human-readable
      "mention_count": 375,                                    // int
      "mention_share": 0.1875,                                 // float
      "quality_score": 0.9752,                                 // float, 0.0–1.0
      "representative_texts": [                                // string[], max 3 sample texts
        "cetaphil oily dùng da nhạy cảm thấy ổn áp ghê",
        "không bị không kích ứng",
        "nếu đang treatment thì thử La Roche-Posay Effaclar đi"
      ]
    }
    // ... max 10 items
  ],
  
  // ── Top Issues (max 10, sorted by issue_pressure_proxy DESC) ──
  "top_issues": [
    {
      "issue_category": "fake_authenticity_concern",           // string
      "mention_count": 200,                                    // int
      "mention_prevalence_ratio": 0.1,                         // float
      "issue_pressure_proxy": 149.477,                         // float
      "severity_mix": {                                        // object string→float
        "low": 0.6,
        "medium": 0.3,
        "high": 0.1
      }
    }
    // ... max 10 items
  ],

  // ── RAG Hints ──
  "rag_collection": "macro_insights",
  "rag_document_type": "report_digest",
  "should_index": true
}
```

### Embedding Strategy (cho knowledge-srv)

Khi consume message này, knowledge-srv nên:

1. **Tạo 1 document text tổng hợp** từ payload — ví dụ:

```text
Report Digest for domain-facial-cleanser-vn (tiktok, 2000 mentions).
Window: 2026-02-01 to 2026-02-10.
Top brands: Cetaphil (11.7%), CeraVe (11.6%), Hada Labo (10.9%).
Top topics: Cleanser Recommendation by Skin Type (18.7%), Oily and Acne-Prone Skin Cleansers (18.5%).
Top issues: fake_authenticity_concern (pressure: 149.5), breakout_after_use (pressure: 120.3).
```

1. **Metadata payload**: Toàn bộ structured data (`top_entities`, `top_topics`, `top_issues`) lưu vào Qdrant metadata để filter/facet.
2. **Point ID format**: `digest:{run_id}`
3. **TTL/Overwrite**: Upsert — mỗi run mới thay thế digest cũ cùng `project_id`.

---

## So Sánh 3 Contracts

| Thuộc tính | `batch.completed` (existing) | `insights.published` (new) | `report.digest` (new) |
| :--- | :--- | :--- | :--- |
| **Grain** | 1 message/batch | 1 message/insight card | 1 message/run |
| **Số lượng/run** | 1 | 5–15 cards | 1 |
| **Nội dung** | Chỉ metadata (file_url) | Narrative text + metrics | Curated top-N summary |
| **RAG value** | Gián tiếp (tải file rồi index) | Rất cao (synthesized knowledge) | Cao (structured overview) |
| **Qdrant collection** | Per-project collection | `macro_insights` | `macro_insights` |
| **Knowledge-srv action** | Download JSONL → embed per-post | Embed title+summary trực tiếp | Build summary text → embed |

---

## Error Handling

| Scenario | Hành vi mong đợi |
|---|---|
| Decode JSON fail | Log error, skip message, commit offset |
| `should_index = false` | Skip indexing, commit offset |
| `contract_version` không nhận diện | Log warning, skip message |
| Qdrant upsert fail | Retry 3 lần (exponential backoff), rồi publish lên DLQ `analytics.insights.published.dlq` / `analytics.report.digest.dlq` |
| Duplicate `run_id` + `insight_type` | Upsert (overwrite point cũ) |

---

## References

- [output_payload.md](file:///Users/tailung/Workspaces/smap-hcmut/analysis-srv/documents/output_payload.md) — Existing analysis-srv output spec
- [type.go](file:///Users/tailung/Workspaces/smap-hcmut/knowledge-srv/internal/indexing/delivery/kafka/type.go) — Existing `BatchCompletedMessage` struct
- [handler.go](file:///Users/tailung/Workspaces/smap-hcmut/knowledge-srv/internal/consumer/handler.go) — Existing consumer domain setup
- [insight_message.py](file:///Users/tailung/Workspaces/smap-hcmut/analysis-srv/internal/model/insight_message.py) — Existing per-post `InsightMessage` model
