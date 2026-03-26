# Master Plan: Knowledge-srv 3-Layer Migration

> **Date:** 2026-03-24
> **Status:** Draft
> **Scope:** Migrate knowledge-srv from legacy flat `AnalyticsPost` model to 3-layer architecture consuming new analysis-srv contracts

---

## 1. Bối cảnh: Tại sao cần migrate

### Legacy hiện tại có gì

knowledge-srv hiện chỉ có **1 consumer duy nhất**: `analytics.batch.completed`.

Flow hiện tại:

```
Kafka (analytics.batch.completed)
  → parse documents[] array thành []AnalyticsPost (flat struct, legacy)
  → embed content field
  → upsert vào Qdrant collection per-project
```

**Vấn đề cốt lõi:** Struct `AnalyticsPost` trong `internal/indexing/types.go` là **legacy flat format**, KHÔNG match với `InsightMessage` mà analysis-srv thực sự produce. Ví dụ:

| Legacy `AnalyticsPost`             | New `InsightMessage` (from analysis-srv)   |
|------------------------------------|--------------------------------------------|
| `ID` (flat)                        | `identity.uap_id` (nested)                 |
| `Content` (flat string)            | `content.clean_text` (nested)              |
| `OverallSentiment` (flat)          | `nlp.sentiment.label` (nested)             |
| `IsSpam`, `IsBot` (flat booleans)  | `rag: bool` — index gate                   |
| Không có entity nested             | `nlp.entities[]` (structured)              |
| Không có impact/priority           | `business.impact.*` (nested)               |

→ knowledge-srv **không thể consume đúng** output mới của analysis-srv nếu không migrate struct.

Ngoài ra, knowledge-srv hoàn toàn **thiếu 2 layer macro-level**:

- Không có consumer cho insight cards (Layer 2)
- Không có consumer cho report digest (Layer 1)
- Không có Qdrant collection `macro_insights`
- reAct agent chỉ search được per-post, không search được narrative/overview

---

## 2. Kiến trúc đích: 3 Layer

```
                    analysis-srv (Python)
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
   Layer 3             Layer 2            Layer 1
analytics.batch.   analytics.insights.  analytics.report.
  completed          published            digest
        │                  │                  │
        └──────────────────┼──────────────────┘
                           │
                    knowledge-srv (Go)
                           │
                ┌──────────┼──────────┐
                │          │          │
             Qdrant     Qdrant     Qdrant
           proj_{id}   macro_     macro_
           (per-post)  insights   insights
                       (cards)   (digest)
```

### Layer 1 — Campaign Overview

- **Trả lời:** "Campaign này như thế nào tổng quan?"
- **Nguồn:** `analytics.report.digest`
- **Collection:** `macro_insights`, filter `rag_document_type = "report_digest"`
- **Cardinality:** 1 document per run per project
- **Semantic value:** Knowledge-srv tự build prose từ structured data

### Layer 2 — Narrative Insights

- **Trả lời:** "Tại sao? Có gì đáng chú ý?"
- **Nguồn:** `analytics.insights.published`
- **Collection:** `macro_insights`, filter `rag_document_type = "insight_card"`
- **Cardinality:** 5–15 cards per run
- **Semantic value:** CAO NHẤT — analysis-srv đã "tiêu hóa" thành narrative

### Layer 3 — Per-post Evidence

- **Trả lời:** "Cho xem bài viết cụ thể / trích dẫn / số liệu chi tiết"
- **Nguồn:** `analytics.batch.completed` (giữ nguyên topic)
- **Collection:** `proj_{project_id}` (per-project, giữ nguyên)
- **Cardinality:** 2000+ documents per batch
- **Semantic value:** Raw evidence, cần migrate struct sang `InsightMessage`

---

## 2.1. Mối quan hệ giữa 3 Layer — Ví dụ cụ thể

### InsightMessage là gì?

**InsightMessage KHÔNG CHỈ là "1 bài post".** Nó đại diện cho **bất kỳ document nào** trong batch:

- **POST** — bài đăng gốc (depth=0), e.g. `tt_p_fc_0001`
- **COMMENT** — comment trực tiếp dưới post (depth=1), e.g. `tt_c_fc_0001_02`
- **REPLY** — reply trả lời comment (depth=2), e.g. `tt_r_fc_0001_03`

Mỗi element trong `documents[]` = 1 InsightMessage = 1 Qdrant point. Knowledge-srv embed **raw text** (`content.clean_text`) — KHÔNG phải summary — vì khi user hỏi "cho xem bài viết cụ thể", cần trả về **nội dung gốc** để trích dẫn.

`content.summary` chỉ là metadata snippet ngắn để hiển thị nhanh trong kết quả search, KHÔNG dùng để embed.

### Layer 2 insight cards đến từ đâu?

7 insight cards KHÔNG sinh ra từ từng post riêng lẻ. Chúng đến từ **pipeline aggregation** của analysis-srv:

```text
2000 documents (posts + comments + replies)
  │
  ├─ NER pipeline ──────────────────→ entity facts
  ├─ Sentiment pipeline ────────────→ sentiment facts
  ├─ Topic pipeline ────────────────→ topic facts
  ├─ Issue pipeline ────────────────→ issue facts
  ├─ Thread analysis ───────────────→ thread facts
  │
  ▼ (aggregate)
  BI Report Pipeline
  ├── sov_report           → share_of_voice_shift card
  ├── buzz_report (entity) → (data cho digest, không sinh card riêng)
  ├── buzz_report (topic)  → trending_topic card + conversation_hotspot card
  ├── emerging_topics      → emerging_topic card
  ├── top_issues           → issue_warning card
  ├── thread_controversy   → controversy_alert card
  └── creator_breakdown    → creator_concentration card
                                     ↓
                              7 insight cards
                        (mỗi card reference ngược lại
                         các uap_id cụ thể qua evidence_references[])
```

Nguồn verify: `outputSRM/insights/insights.jsonl` chứa đúng 7 cards với 7 `insight_type` khác nhau, mỗi card có `evidence_references` trỏ về các `uap_id` trong Layer 3.

### Cardinality

```text
1 Project (project_id, campaign_id)
  └── 1 Analysis Run (run_id)
        │
        ├── Layer 1:  1 Kafka message  (analytics.report.digest)        ← 1:1 với run
        │             → 1 Qdrant point
        │
        ├── Layer 2:  7 Kafka messages (analytics.insights.published)   ← 1:N với run (N = số report types)
        │             → 7 Qdrant points
        │
        └── Layer 3:  1 Kafka message  (analytics.batch.completed)      ← 1:1 với run (Direct array)
                      └── documents[] chứa 2000 InsightMessage (posts + comments + replies)
                      → 2000 Qdrant points
```

**`run_id`** là key liên kết chung giữa cả 3 layer.

### Ví dụ end-to-end: Project "Sữa rửa mặt Q1 2026"

#### Bước 0 — Crawl & gửi sang Analysis

Crawler thu thập **2000 documents** TikTok về sữa rửa mặt (bao gồm posts + comments + replies), gán vào:

- `project_id` = `"proj_cleanser_01"`
- `campaign_id` = `"camp_q1_2026"` (bắt buộc)

Cấu trúc UAP input (verify từ `uap-batches/` sample):

```json
// POST (depth=0) — tt_p_fc_0001
{"identity": {"uap_id": "tt_p_fc_0001", "uap_type": "POST", "platform": "tiktok"},
 "hierarchy": {"parent_id": null, "root_id": "tt_p_fc_0001", "depth": 0},
 "content": {"text": "cetaphil oily dùng da nhạy cảm thấy ổn áp ghê, rửa sạch mà không bị kích ứng..."},
 "engagement": {"likes": 2837, "comments_count": 13, "shares": 27, "views": 25512}}

// COMMENT (depth=1) — tt_c_fc_0001_02, reply cho post trên
{"identity": {"uap_id": "tt_c_fc_0001_02", "uap_type": "COMMENT", "platform": "tiktok"},
 "hierarchy": {"parent_id": "tt_p_fc_0001", "root_id": "tt_p_fc_0001", "depth": 1},
 "content": {"text": "ceta oily trên da mình bị rít da luôn 😩 super gentle"}}

// REPLY (depth=2) — tt_r_fc_0001_03, reply cho comment
{"identity": {"uap_id": "tt_r_fc_0001_03", "uap_type": "REPLY", "platform": "tiktok"},
 "hierarchy": {"parent_id": "tt_c_fc_0001_03", "root_id": "tt_p_fc_0001", "depth": 2},
 "content": {"text": "same here, nhat la khi da đang mụn ẩn + dầu vùng T"}}
```

Data gửi sang analysis-srv. Analysis chạy full pipeline: NER → sentiment → aspect → topic → issue → thread analysis → BI reports → insight generation. Khi xong, publish **3 loại message** về Kafka.

#### Bước 1 — Knowledge-srv nhận Layer 3 (per-document evidence)

Analysis-srv publish **1 Kafka message** vào topic `analytics.batch.completed`, trong đó `documents[]` chứa toàn bộ 2000 `InsightMessage` object:

```json
{
  "project_id":  "proj_cleanser_01",
  "campaign_id": "camp_q1_2026",
  "documents": [ ... ]
}
```

Knowledge-srv parse `documents[]` trực tiếp → **2000 elements**, mỗi element là 1 `InsightMessage` — gồm cả posts, comments, replies.

Ví dụ: element **POST** `tt_p_fc_0001`:

```json
{
  "identity": {
    "uap_id": "tt_p_fc_0001",
    "uap_type": "post",
    "uap_media_type": "video",
    "platform": "TIKTOK",
    "published_at": "2026-02-01T10:17:00Z"
  },
  "content": {
    "clean_text": "cetaphil oily dùng da nhạy cảm thấy ổn áp ghê, rửa sạch mà không bị kích ứng",
    "summary": "User khen Cetaphil cho da nhạy cảm"
  },
  "nlp": {
    "sentiment": { "label": "POSITIVE", "score": 0.72 },
    "aspects": [{ "aspect": "GENTLE", "polarity": "POSITIVE" }],
    "entities": [{ "type": "BRAND", "value": "Cetaphil" }]
  },
  "business": {
    "impact": { "engagement": {"likes": 2837, "comments": 13, "shares": 27, "views": 25512}, "impact_score": 72.5, "priority": "HIGH" }
  },
  "rag": true
}
```

Ví dụ: element **COMMENT** `tt_c_fc_0001_02` (comment dưới post trên):

```json
{
  "identity": {
    "uap_id": "tt_c_fc_0001_02",
    "uap_type": "comment",
    "uap_media_type": "text",
    "platform": "TIKTOK",
    "published_at": "2026-02-01T10:28:00Z"
  },
  "content": {
    "clean_text": "ceta oily trên da mình bị rít da luôn super gentle",
    "summary": "User phàn nàn Cetaphil Oily bị rít"
  },
  "nlp": {
    "sentiment": { "label": "NEGATIVE", "score": -0.45 },
    "aspects": [{ "aspect": "TEXTURE", "polarity": "NEGATIVE" }],
    "entities": [{ "type": "PRODUCT", "value": "Cetaphil Oily Skin Cleanser" }]
  },
  "business": {
    "impact": { "engagement": {"likes": 88, "comments": 2, "shares": 0, "views": 0}, "impact_score": 15.3, "priority": "LOW" }
  },
  "rag": true
}
```

→ Knowledge-srv:

1. Check `rag == true` cho mỗi dòng
2. Embed `content.clean_text` (**raw text gốc, KHÔNG phải summary**)
3. Upsert **2000 points** vào Qdrant collection `proj_cleanser_01`
4. Mỗi point lưu metadata: sentiment, aspects, entities, engagement, impact_score, priority... để filter/trả kết quả

#### Bước 2 — Knowledge-srv nhận Layer 2 (insight cards)

Analysis-srv, sau khi aggregate toàn bộ 2000 documents qua BI report pipeline, sinh ra **7 insight cards** và publish **7 Kafka messages riêng lẻ** vào topic `analytics.insights.published`.

Mỗi card = 1 message. Card đến từ report nào:

| insight_type | Sinh từ BI report nào | Ý nghĩa |
|---|---|---|
| `share_of_voice_shift` | `sov_report` | Brand tăng/giảm thị phần trong window |
| `trending_topic` | `buzz_report` (topic section) | Topic đang tăng volume |
| `conversation_hotspot` | `buzz_report` (topic section) | Topic có buzz score cao |
| `emerging_topic` | `emerging_topics_report` | Topic mới xuất hiện lần đầu |
| `issue_warning` | `top_issues_report` | Issue có áp lực cao |
| `controversy_alert` | `thread_controversy_report` | Thread tranh cãi |
| `creator_concentration` | `creator_source_breakdown_report` | Tập trung tác giả bất thường |

**Message 1/7** (verify từ `outputSRM/insights/insights.jsonl` dòng 1):

```json
{
  "project_id": "proj_cleanser_01",
  "campaign_id": "camp_q1_2026",
  "run_id": "run-20260323T165146Z",
  "insight_type": "share_of_voice_shift",
  "title": "Share of voice shift detected for Cetaphil",
  "summary": "Cetaphil lost 22 mention(s) in the later half of the window.",
  "confidence": 0.5612,
  "analysis_window_start": "2026-02-01T10:17:00Z",
  "analysis_window_end": "2026-02-10T21:24:00Z",
  "supporting_metrics": { "mention_share": 0.117, "delta_mention_share": -0.0112 },
  "evidence_references": ["tt_p_fc_0001", "tt_c_fc_0001_02", "tt_c_fc_0001_03", "tt_c_fc_0001_06", "tt_c_fc_0001_09"],
  "should_index": true
}
```

Chú ý `evidence_references`: trỏ về `tt_p_fc_0001` (POST), `tt_c_fc_0001_02` (COMMENT)... — đây là cross-reference sang Layer 3. Khi user hỏi "tại sao Cetaphil mất thị phần?", agent trả insight card, rồi drill-down sang Layer 3 lấy nội dung gốc từ các uap_id này.

→ Knowledge-srv embed `"{title}. {summary}"` cho mỗi card → upsert **7 points** vào collection `macro_insights`.

#### Bước 3 — Knowledge-srv nhận Layer 1 (report digest)

Analysis-srv publish **1 Kafka message** vào topic `analytics.report.digest`. Data tổng hợp từ toàn bộ BI reports (verify cấu trúc từ `outputSRM/reports/bi_reports.json`):

```json
{
  "project_id": "proj_cleanser_01",
  "campaign_id": "camp_q1_2026",
  "run_id": "run-20260323T165146Z",
  "analysis_window_start": "2026-02-01T10:17:00Z",
  "analysis_window_end": "2026-02-10T21:24:00Z",
  "domain_overlay": "domain-facial-cleanser-vn",
  "platform": "tiktok",
  "total_mentions": 2000,
  "top_entities": [
    { "canonical_entity_id": "brand.cetaphil", "entity_name": "Cetaphil", "entity_type": "brand", "mention_count": 234, "mention_share": 0.117 },
    { "canonical_entity_id": "brand.cerave", "entity_name": "CeraVe", "entity_type": "brand", "mention_count": 233, "mention_share": 0.1165 }
  ],
  "top_topics": [
    { "topic_key": "cleanser_brand_comparison", "topic_label": "Cleanser Brand and Product Comparison", "mention_count": 364, "buzz_score_proxy": 429.37 },
    { "topic_key": "cleanser_recommendation_by_skin_type", "topic_label": "Cleanser Recommendation by Skin Type", "mention_count": 375, "quality_score": 0.9752 }
  ],
  "top_issues": [
    { "issue_category": "fake_authenticity_concern", "mention_count": 200, "issue_pressure_proxy": 149.477, "severity_mix": {"low": 0.6, "medium": 0.3, "high": 0.1} }
  ],
  "should_index": true
}
```

→ Knowledge-srv tự build prose → embed → upsert **1 point** vào `macro_insights`.

#### Tổng kết: Knowledge-srv nhận gì từ 1 run?

| Bước | Kafka topic | Kafka msgs | Qdrant points | Collection | Pattern |
|------|-------------|------------|---------------|------------|---------|
| 1 | `analytics.batch.completed` | 1 | **2000** (from documents[]: posts+comments+replies) | `proj_cleanser_01` | Direct array |
| 2 | `analytics.insights.published` | **7** (1 per insight_type) | **7** | `macro_insights` | Direct |
| 3 | `analytics.report.digest` | **1** | **1** | `macro_insights` | Direct |
| **Tổng** | | **9 Kafka msgs** | **2008 Qdrant points** | 2 collections | |

#### Cross-reference giữa các layer

```text
Layer 1 (digest)                     Layer 2 (cards)                        Layer 3 (documents)
┌──────────────────────┐    run_id   ┌───────────────────────────┐          ┌──────────────────────┐
│ digest:run-202603... │ ◄─────────► │ sov_shift card            │          │ tt_p_fc_0001 (POST)  │
│                      │             │   evidence_references: ───────────►  │ tt_c_fc_0001_02 (CMT)│
│ top_entities[]:      │             │     ["tt_p_fc_0001",      │          │ tt_c_fc_0001_03 (CMT)│
│   Cetaphil: 234 lần  │             │      "tt_c_fc_0001_02"..] │          │ tt_c_fc_0001_06 (CMT)│
│   CeraVe:  233 lần   │             │                           │          │ tt_c_fc_0001_09 (CMT)│
│   (aggregated stats  │             │ trending_topic card       │          │                      │
│    từ 2000 docs)     │             │   evidence_references: ───────────►  │ tt_c_fc_0001_10 (CMT)│
│                      │             │     ["tt_c_fc_0001_10"..] │          │ tt_c_fc_0002_11 (CMT)│
│ top_topics[]:        │             │                           │          │ tt_r_fc_0002_01 (RPL)│
│   Brand Comparison   │             │ issue_warning card        │          │                      │
│   Recommendation     │             │   evidence_references: ───────────►  │ tt_c_fc_0001_06 (CMT)│
│   (aggregated stats) │             │     ["tt_c_fc_0001_06"..] │          │ tt_c_fc_0001_07 (CMT)│
│                      │             │ ...                       │          │ ...                  │
│                      │             │ (7 cards total)           │          │ (2000 docs total)    │
└──────────────────────┘             └───────────────────────────┘          └──────────────────────┘
       1 point                              7 points                            2000 points

Mối liên hệ:
  • Layer 2 → Layer 3: evidence_references[] chứa uap_id trực tiếp
    → agent dùng để drill-down lấy raw content khi user cần trích dẫn
  • Layer 1 ↔ Layer 2: cùng run_id + project_id
    → Layer 1 cho bức tranh tổng, Layer 2 giải thích "tại sao"
  • Layer 1 → Layer 3: top_entities[].canonical_entity_id match entities trong các docs
    → aggregated stats (234 mentions) tính từ toàn bộ 2000 documents
```

---

## 3. Kafka Queues — Danh sách chính thức

### Queues mà knowledge-srv PHẢI consume

| # | Topic                          | Consumer Group                | Format       | Msgs/run | Layer | Action                         |
|---|--------------------------------|-------------------------------|--------------|----------|-------|--------------------------------|
| 1 | `analytics.batch.completed`    | `knowledge-indexing-batch`    | JSON object  | 1        | 3     | Parse documents[] → embed per-document |
| 2 | `analytics.insights.published` | `knowledge-indexing-insights` | JSON object  | 5–15     | 2     | Embed title+summary trực tiếp  |
| 3 | `analytics.report.digest`      | `knowledge-indexing-digest`   | JSON object  | 1        | 1     | Build prose → embed            |

### DLQ topics (knowledge-srv publish khi retry exhausted)

| # | Topic                              | Trigger                          |
|---|------------------------------------|----------------------------------|
| 4 | `analytics.insights.published.dlq` | Qdrant upsert fail sau 3 retries |
| 5 | `analytics.report.digest.dlq`      | Qdrant upsert fail sau 3 retries |
| 6 | `analytics.batch.completed.dlq`    | Giữ nguyên logic DLQ hiện tại    |

### Legacy cần REMOVE trong refactor này

| # | Item | Loại | Vị trí hiện tại | Action |
|---|------|------|------------------|--------|
| 1 | `smap.analytics.output` topic | Kafka topic (chưa bao giờ implement) | Chỉ tồn tại trong docs (`analysis_knowledge_contracts.md`, `hint.md`) | Xóa khỏi docs, KHÔNG tạo consumer cho topic này |
| 2 | `smap_analytics` collection (hardcoded) | Qdrant collection | `internal/point/repository/qdrant/point.go:12` — `const collectionName = "smap_analytics"` | Thay bằng dynamic collection name: `proj_{project_id}` cho Layer 3, `macro_insights` cho Layer 1+2. Cần refactor `UpsertOptions` và `SearchOptions` để nhận `collectionName` param thay vì dùng const |
| 3 | `AnalyticsPost` struct (legacy flat) | Go struct | `internal/indexing/types.go` | Replace bằng `InsightMessage` (nested, đúng contract) |

---

## 4. Payload Contracts — Chi tiết từng Layer

### Messaging Pattern: Cả 3 layer dùng Direct Payload

Tất cả 3 layer đều dùng **Direct payload** — data nằm trực tiếp trong Kafka message, không qua MinIO:

| Layer | Pattern | Ghi chú |
|-------|---------|----------|
| **Layer 3** | **Direct payload** (`documents[]` array) | Payload lớn (~4MB với 2000 docs). **Bắt buộc** config Kafka `max.message.bytes` ≥ 10MB trên cả broker, producer (analysis-srv), và consumer (knowledge-srv). |
| **Layer 2** | **Direct payload** (1 card/message) | ~500 bytes/message. 5–15 messages/run. Nhỏ gọn. |
| **Layer 1** | **Direct payload** (1 digest/run) | ~2–5KB/message. Nhỏ gọn. |

```text
Layer 3 (Direct array):
  Kafka msg: { project_id, campaign_id, documents: [{...InsightMessage...}, ...] }
                  ↑ payload đầy đủ, cần Kafka max.message.bytes ≥ 10MB

Layer 2 & 1 (Direct):
  Kafka msg: { title, summary, ... }  /  { top_entities, top_topics, ... }
                  ↑ payload đầy đủ, nhỏ gọn
```

### 4.1. Layer 3: `analytics.batch.completed`

#### Kafka message (Direct array payload)

**CẦN THAY ĐỔI:** Format mới bỏ `file_url`, đưa thẳng `documents[]` array vào Kafka message.

```json
{
  "project_id":  "proj_cleanser_01",
  "campaign_id": "camp_q1_2026",
  "documents": [
    { "identity": {...}, "content": {...}, "nlp": {...}, "business": {...}, "rag": true },
    ...
  ]
}
```

> **Cấu hình bắt buộc:** `max.message.bytes` ≥ 10MB trên Kafka broker, analysis-srv producer, và knowledge-srv consumer.

**CẦN THAY ĐỔI:**

`BatchCompletedMessage` struct: chỉ giữ `project_id`, `campaign_id`, `documents []InsightMessage`. Bỏ các field `batch_id`, `record_count`, `completed_at`. Parse từng element thành `InsightMessage` (nested, đúng contract) thay vì `AnalyticsPost` (flat, legacy).

#### InsightMessage struct mới — Fields knowledge-srv cần consume

```
InsightMessage
├── identity
│   ├── uap_id: string               // *** primary key, cross-ref from Layer 2
│   ├── uap_type: string             // "post" | "comment" | "reply"
│   ├── uap_media_type: string       // "video" | "image" | "carousel" | "text" | "live" | "other"
│   ├── platform: string             // "TIKTOK" → metadata platform filter
│   └── published_at: ISO8601        // *** time filter
├── content
│   ├── clean_text: string           // *** TEXT ĐỂ EMBED
│   └── summary: string              // display snippet trong search results
├── nlp
│   ├── sentiment
│   │   ├── label: string            // *** POSITIVE|NEGATIVE|NEUTRAL|MIXED → filter
│   │   └── score: float             // -1.0 to 1.0 → rank
│   ├── aspects[]
│   │   ├── aspect: string           // *** filter by aspect
│   │   └── polarity: string
│   └── entities[]
│       ├── type: string             // PRODUCT|BRAND|PERSON|LOCATION
│       └── value: string            // *** filter by entity
├── business
│   └── impact
│       ├── engagement: {likes, comments, shares, views}
│       ├── impact_score: float      // 0–100 → rank
│       └── priority: string         // *** LOW|MEDIUM|HIGH|CRITICAL → filter
└── rag: bool                        // *** GATE CHÍNH — true → index, false → skip
```

**Qdrant Point cho Layer 3:**

- **Collection:** `proj_{project_id}`
- **Point ID:** `identity.uap_id`
- **Embed text:** `content.clean_text`
- **Metadata (không embed, lưu payload):**
  - `project_id`, `campaign_id`
  - `platform`, `uap_id`, `uap_type`, `uap_media_type`, `published_at`
  - `sentiment_label`, `sentiment_score`
  - `aspects[]` (aspect key + polarity)
  - `entities[]` (type + value)
  - `impact_score`, `priority`
  - `content_summary` (snippet)

---

### 4.2. Layer 2: `analytics.insights.published`

**Kafka message — 1 object per insight card:**

```json
{
  "project_id": "proj_cleanser_01",
  "campaign_id": "camp_q1_2026",
  "run_id": "run-20260323T165146Z",

  "insight_type": "share_of_voice_shift",
  "title": "Share of voice shift detected for Cetaphil",
  "summary": "Cetaphil lost 22 mention(s) in the later half of the window.",
  "confidence": 0.5612,

  "analysis_window_start": "2026-02-01T10:17:00Z",
  "analysis_window_end": "2026-02-10T21:24:00Z",

  "supporting_metrics": { "mention_share": 0.117, "delta_mention_share": -0.0112 },
  "evidence_references": ["tt_p_fc_0001", "tt_c_fc_0001_02"],
  "should_index": true
}
```

**`insight_type` enum values:**

- `share_of_voice_shift` — brand tăng/giảm thị phần
- `trending_topic` — topic đang tăng volume
- `conversation_hotspot` — topic có buzz score cao
- `emerging_topic` — topic mới xuất hiện lần đầu
- `issue_warning` — issue có áp lực cao
- `controversy_alert` — thread tranh cãi
- `creator_concentration` — tập trung tác giả bất thường

**knowledge-srv processing:**

1. Check `should_index == true`
2. Embed text = `"{title}. {summary}"` (chỉ cần concat, không thêm gì)
3. Upsert vào Qdrant collection `macro_insights` (hằng số — derived từ Kafka topic)

**Qdrant Point cho Layer 2:**

- **Collection:** `macro_insights`
- **Point ID:** `insight:{run_id}:{insight_type}:{sha256(title)[:12]}`
- **Embed text:** `"{title}. {summary}"`
- **Metadata payload:**
  - `project_id`, `campaign_id`, `run_id`
  - `rag_document_type` = `"insight_card"` (knowledge-srv tự gán, không từ payload)
  - `insight_type`
  - `confidence`
  - `analysis_window_start`, `analysis_window_end`
  - `supporting_metrics` (raw JSON object)
  - `evidence_references` (string array — cross-link sang Layer 3)
- **Upsert strategy:** Khi run mới, insight cùng `project_id` + `insight_type` bị thay thế

---

### 4.3. Layer 1: `analytics.report.digest`

**Kafka message — 1 object per run:**

```json
{
  "project_id": "proj_cleanser_01",
  "campaign_id": "camp_q1_2026",
  "run_id": "run-20260323T165146Z",

  "analysis_window_start": "2026-02-01T10:17:00Z",
  "analysis_window_end": "2026-02-10T21:24:00Z",

  "domain_overlay": "domain-facial-cleanser-vn",
  "platform": "tiktok",
  "total_mentions": 2000,

  "top_entities": [
    {
      "canonical_entity_id": "brand.cetaphil",
      "entity_name": "Cetaphil",
      "entity_type": "brand",
      "mention_count": 234,
      "mention_share": 0.117
    }
  ],
  "top_topics": [
    {
      "topic_key": "cleanser_recommendation_by_skin_type",
      "topic_label": "Cleanser Recommendation by Skin Type",
      "mention_count": 375,
      "mention_share": 0.1875,
      "quality_score": 0.9752,
      "representative_texts": ["sample 1", "sample 2"]
    }
  ],
  "top_issues": [
    {
      "issue_category": "fake_authenticity_concern",
      "mention_count": 200,
      "issue_pressure_proxy": 149.477,
      "severity_mix": { "low": 0.6, "medium": 0.3, "high": 0.1 }
    }
  ],

  "should_index": true
}
```

**knowledge-srv processing:**

1. **Gate check:** If `should_index == false` → skip entirely (no indexing, no NotebookLM export)

2. **Prose generation** — Build coherent narrative text from structured data

   **Pseudo-code:**

   ```golang
   prose := ""

   // Section 1: Report Header
   prose += fmt.Sprintf("Campaign Report: %s\n", domainOverlay)
   prose += fmt.Sprintf("Platform: %s | Mentions: %d\n", platform, totalMentions)
   prose += fmt.Sprintf("Analysis Window: %s to %s\n", windowStart, windowEnd)
   prose += "\n"

   // Section 2: Top Entities/Brands (max 5)
   prose += "Top Brands:\n"
   for i := 0; i < min(5, len(topEntities)); i++ {
     entity := topEntities[i]
     prose += fmt.Sprintf("- %s: %d mentions (%.1f%% share)\n",
       entity.Name, entity.MentionCount, entity.MentionShare*100)
   }
   prose += "\n"

   // Section 3: Top Topics (max 5)
   prose += "Key Discussion Topics:\n"
   for i := 0; i < min(5, len(topTopics)); i++ {
     topic := topTopics[i]
     prose += fmt.Sprintf("- %s: %d mentions", topic.Label, topic.MentionCount)
     if topic.QualityScore > 0 {
       prose += fmt.Sprintf(" (quality: %.2f)", topic.QualityScore)
     }
     if len(topic.RepresentativeTexts) > 0 {
       prose += fmt.Sprintf("\n  Example: \"%s\"", topic.RepresentativeTexts[0])
     }
     prose += "\n"
   }
   prose += "\n"

   // Section 4: Top Issues (max 5)
   prose += "Critical Issues:\n"
   for i := 0; i < min(5, len(topIssues)); i++ {
     issue := topIssues[i]
     prose += fmt.Sprintf("- %s: %d mentions (pressure: %.2f)\n",
       issue.Category, issue.MentionCount, issue.IssuePressureProxy)
   }
   ```

   **Example output (for reference):**

   ```text
   Campaign Report: domain-facial-cleanser-vn
   Platform: tiktok | Mentions: 2000
   Analysis Window: 2026-02-01T10:17:00Z to 2026-02-10T21:24:00Z

   Top Brands:
   - Cetaphil: 234 mentions (11.7% share)
   - CeraVe: 233 mentions (11.7% share)
   - Hada Labo: 218 mentions (10.9% share)

   Key Discussion Topics:
   - Cleanser Brand Comparison: 364 mentions (quality: 0.98)
     Example: "Dùng thử 5 loại sữa rửa mặt..."
   - Recommendation by Skin Type: 375 mentions
     Example: "Da dầu nên dùng gì?"

   Critical Issues:
   - fake_authenticity_concern: 200 mentions (pressure: 149.48)
   ```

3. **Embedding** — Send generated prose to embedding model
   - Input: complete prose text (all sections concatenated)
   - Output: vector → upsert to Qdrant

4. **Metadata** — Store structured data separately (see Qdrant point structure below)

**Qdrant Point Structure cho Layer 1:**

```json
{
  "collection": "macro_insights",
  "point_id": "digest:{run_id}",
  "vector": [embed(generated_prose)],
  "payload": {
    // Envelope scoping
    "project_id": "proj_cleanser_01",
    "campaign_id": "camp_q1_2026",
    "run_id": "run-20260323T165146Z",

    // Document type identifier (hardcoded by knowledge-srv)
    "rag_document_type": "report_digest",

    // Window & domain context
    "analysis_window_start": "2026-02-01T10:17:00Z",
    "analysis_window_end": "2026-02-10T21:24:00Z",
    "domain_overlay": "domain-facial-cleanser-vn",
    "platform": "tiktok",
    "total_mentions": 2000,

    // Full structured data (for filtering & display)
    "top_entities": [...],     // array of 5-10 entities
    "top_topics": [...],       // array of 5-10 topics
    "top_issues": [...]        // array of 5-10 issues
  }
}
```

**Details:**

- **Collection:** `macro_insights` (shared with Layer 2 insights)
- **Point ID format:** `digest:{run_id}` (e.g., `digest:run-20260323T165146Z`)
  - Unique per run per project (run_id is globally unique)
  - Enables upsert strategy: new run with same run_id replaces old digest

- **Vector:** Generated prose text embedded
  - Input to embedding model: complete multi-section prose (header + brands + topics + issues)
  - Output: single vector for Qdrant

- **Payload fields:**
  - **Identifiers:** project_id, campaign_id, run_id (for scoping & filtering)
  - **Document type:** rag_document_type = "report_digest" (hardcoded by knowledge-srv to distinguish from insight_card)
  - **Context:** analysis_window_start/end, domain_overlay, platform, total_mentions
  - **Structured data:** top_entities[], top_topics[], top_issues[] stored as-is for drill-down queries

- **Upsert strategy:**
  - Check if point with ID `digest:{run_id}` already exists
  - If yes: replace (overwrite entire payload + vector)
  - If no: create new
  - Effect: latest digest per run always available; old digests from same run replaced

---

## 4.4. NotebookLM Export — 1 Campaign = 1 Notebook

### Mục tiêu

Ngoài Qdrant indexing để phục vụ reAct agent, knowledge-srv cũng cần **export dữ liệu đã xử lý sang NotebookLM** để user có thể tương tác bằng giao diện Google AI Studio. Mỗi campaign → 1 notebook riêng biệt.

### Trigger

Knowledge-srv lắng nghe `analytics.report.digest` (Layer 1). Sau khi index thành công digest vào Qdrant, cùng `run_id` đó được dùng làm trigger cho export pipeline:

```text
analytics.report.digest consumed
  └── IndexDigest() → Qdrant ✓
  └── ExportToNotebookLM() → Google Drive ✓  ← NEW
```

Tại sao dùng digest làm trigger thay vì batch.completed?

- Digest là message cuối cùng được publish sau khi analysis-srv hoàn thành toàn bộ pipeline
- Layer 2 và Layer 3 đã được index trước đó → khi digest đến, toàn bộ 3 layer đã có trong Qdrant
- Digest chứa `run_id`, `campaign_id`, `project_id` đủ để fetch và assemble tất cả data

### 1 Campaign = 1 Notebook

```text
Google Drive
└── smap-notebooks/
    └── {campaign_id}/                          ← 1 thư mục per campaign
        ├── 00_overview_{run_id}.md             ← Layer 1 (digest prose)
        ├── 01_insights_{run_id}.md             ← Layer 2 (all insight cards)
        └── 02_evidence_{run_id}.md             ← Layer 3 (top-N evidence posts)
```

NotebookLM được cấu hình để monitor thư mục `smap-notebooks/{campaign_id}/`. Mỗi lần có run mới, 3 files mới được upload → notebook tự động cập nhật.

> **Note:** Nếu dùng NotebookLM API trực tiếp thay vì qua Drive, notebook_id được map từ `campaign_id` và lưu vào config.

### File format per run

#### `00_overview_{run_id}.md` — Layer 1

```markdown
# Campaign Overview: {domain_overlay}
**Platform:** {platform} | **Window:** {analysis_window_start} → {analysis_window_end}
**Total Mentions:** {total_mentions}

## Share of Voice — Top Brands
| Brand | Mentions | Share | Trend |
|-------|----------|-------|-------|
| Cetaphil | 234 | 11.7% | ↓ -22 |
| CeraVe   | 233 | 11.7% | ↑ +37 |

## Top Discussion Topics
| Topic | Mentions | Quality |
|-------|----------|---------|
| Cleanser Brand Comparison | 364 | — |
| Recommendation by Skin Type | 375 | 0.975 |

## Top Issues
| Issue | Mentions | Pressure |
|-------|----------|----------|
| fake_authenticity_concern | 200 | 149.5 |
```

#### `01_insights_{run_id}.md` — Layer 2

```markdown
# Narrative Insights — Run {run_id}
*Generated from {total_insight_count} insight cards*

---

## [share_of_voice_shift] Share of voice shift detected for Cetaphil
**Confidence:** 0.56 | **Source:** sov_report
Cetaphil lost 22 mention(s) in the later half of the window.
*Evidence:* tt_p_fc_0001, tt_c_fc_0001_02, tt_c_fc_0001_03

---

## [trending_topic] Topic gaining volume: Cleanser Brand Comparison
**Confidence:** 0.80 | **Source:** buzz_report
...
```

#### `02_evidence_{run_id}.md` — Layer 3 (sampled)

Không dump toàn bộ 2000 records — chỉ lấy **top-N** (e.g. top 50) theo `impact_score` hoặc priority = HIGH/CRITICAL:

```markdown
# Top Evidence Posts — Run {run_id}
*Showing top 50 documents by impact score (filtered from {total_count} total)*

---

### [POST] tt_p_fc_0001 — Routine Chân Thật (2026-02-01)
**Sentiment:** POSITIVE (0.72) | **Impact:** 72.5 (HIGH)
**Entities:** Cetaphil | **Aspects:** GENTLE (positive)
> "cetaphil oily dùng da nhạy cảm thấy ổn áp ghê, rửa sạch mà không bị kích ứng"
*Source: TikTok — https://www.tiktok.com/@cleanserlab/video/8800000001*

---

### [COMMENT] tt_c_fc_0001_02 — routine basic (2026-02-01)
**Sentiment:** NEGATIVE (-0.45) | **Impact:** 15.3 (LOW)
**Entities:** Cetaphil Oily Skin Cleanser | **Aspects:** TEXTURE (negative)
> "ceta oily trên da mình bị rít da luôn super gentle"
...
```

### Codebase additions

```text
internal/notebooklm/                         ← NEW package
├── export.go                                 ← ExportToNotebookLM(ctx, ExportInput) error
├── formatter/
│   ├── digest.go                             ← buildOverviewFile(digest ReportDigestMessage) string
│   ├── insights.go                           ← buildInsightsFile(cards []InsightsPublishedMessage) string
│   └── evidence.go                           ← buildEvidenceFile(docs []InsightMessage, topN int) string
├── drive/
│   └── uploader.go                           ← UploadToDrive(ctx, folder, filename, content) error
└── types.go                                  ← ExportInput, ExportConfig
```

`ExportInput`:

```go
type ExportInput struct {
    RunID      string
    CampaignID string
    ProjectID  string
    Digest     ReportDigestMessage          // Layer 1
    Insights   []InsightsPublishedMessage   // Layer 2 — all cards for this run
    TopN       int                          // How many posts to include (default 50)
}
```

Knowledge-srv lấy Layer 2 data bằng cách: sau khi IndexDigest thành công, query Qdrant `macro_insights` với filter `run_id == current_run_id && rag_document_type == "insight_card"` để lấy tất cả 7 cards, rồi fetch top-N posts từ `proj_{project_id}` filter `run_id`.

### Migration checklist cho NotebookLM

Thêm vào Section 11 Validation Checklist:

- [ ] `analytics.report.digest` consumed → 3 files xuất hiện trong Google Drive folder `smap-notebooks/{campaign_id}/`
- [ ] File `00_overview` chứa đúng `domain_overlay`, `total_mentions`, top entities/topics/issues
- [ ] File `01_insights` chứa đủ 7 insight cards từ run hiện tại
- [ ] File `02_evidence` chứa top-50 posts sorted by impact_score, KHÔNG duplicate từ run trước
- [ ] Run mới → 3 files mới được thêm (files cũ vẫn giữ lại để NotebookLM có historical context)
- [ ] Drive upload fail → retry 3× → log error (KHÔNG publish DLQ, export là best-effort)

---

## 5. Qdrant Collection Strategy

| Collection        | Scope           | Document types                          | Dùng khi nào                        |
|-------------------|-----------------|-----------------------------------------|--------------------------------------|
| `macro_insights`  | Global (shared) | `report_digest` (L1), `insight_card` (L2) | Câu hỏi tổng quan, narrative, tại sao |
| `proj_{project_id}` | Per-project  | Per-post evidence (L3)                  | Trích dẫn, số liệu cụ thể, bài viết  |

### Metadata indexes cần tạo trên `macro_insights`

- `project_id` (keyword) — scope filter bắt buộc
- `campaign_id` (keyword)
- `rag_document_type` (keyword) — phân biệt digest vs card
- `insight_type` (keyword) — filter loại insight
- `analysis_window_start` (integer/timestamp)
- `analysis_window_end` (integer/timestamp)
- `confidence` (float) — rank/threshold

### Metadata indexes cần tạo/update trên `proj_{project_id}`

- `project_id` (keyword)
- `campaign_id` (keyword)
- `sentiment_label` (keyword)
- `priority` (keyword)
- `source_type` (keyword)
- `author_type` (keyword)
- `published_at` (integer/timestamp)
- `impact_score` (float)

---

## 6. reAct Agent Query Routing

| Loại câu hỏi                             | Collection đích      | Filter                                    |
|-------------------------------------------|----------------------|-------------------------------------------|
| Tổng quan campaign                        | `macro_insights`     | `rag_document_type=report_digest`         |
| Tại sao? Có gì đáng chú ý?              | `macro_insights`     | `rag_document_type=insight_card`          |
| Narrative + overview kết hợp              | `macro_insights`     | `campaign_id` filter                      |
| Trích dẫn / bài viết cụ thể             | `proj_{project_id}`  | sentiment, entity, aspect filters         |
| Viết report (cần structure + citation)    | **Cả 2 collection** | L1+L2 cho structure, L3 cho citation      |

---

## 7. Error Handling — Thống nhất cho cả 3 Layer

| Scenario                         | Action                                                      |
|----------------------------------|-------------------------------------------------------------|
| JSON decode fail                 | Log error, skip message, commit offset                      |
| `should_index = false`           | Skip indexing, commit offset (không phải error)             |
| `contract_version` không hỗ trợ | Log warning, skip message, commit offset                    |
| Embedding fail                   | Retry 3× exponential backoff → publish DLQ                  |
| Qdrant upsert fail               | Retry 3× exponential backoff → publish DLQ                  |
| Duplicate `run_id` + type        | Upsert (overwrite point cũ) — đây là behavior mong muốn    |

---

## 8. Go Code Structure — Đề xuất tổ chức

### 8.1. Kafka types — `internal/indexing/delivery/kafka/type.go`

```
// Existing (giữ nguyên)
TopicBatchCompleted   = "analytics.batch.completed"
GroupIDBatchCompleted  = "knowledge-indexing-batch"
BatchCompletedMessage struct { ... }

// NEW
TopicInsightsPublished   = "analytics.insights.published"
GroupIDInsightsPublished  = "knowledge-indexing-insights"
InsightsPublishedMessage struct { ... }     // Contract 1 payload

TopicReportDigest   = "analytics.report.digest"
GroupIDReportDigest  = "knowledge-indexing-digest"
ReportDigestMessage struct { ... }          // Contract 2 payload

// DLQ topics
TopicInsightsPublishedDLQ = "analytics.insights.published.dlq"
TopicReportDigestDLQ      = "analytics.report.digest.dlq"
```

### 8.2. InsightMessage struct — `internal/indexing/types.go`

Thay thế `AnalyticsPost` bằng `InsightMessage` (nested struct match analysis-srv contract).
Giữ `AnalyticsPost` alias nếu cần backward compat, nhưng tất cả code mới dùng `InsightMessage`.

### 8.3. Consumer interface — `internal/indexing/delivery/kafka/consumer/new.go`

```go
type Consumer interface {
    ConsumeBatchCompleted(ctx context.Context) error       // Existing
    ConsumeInsightsPublished(ctx context.Context) error    // NEW
    ConsumeReportDigest(ctx context.Context) error         // NEW
    Close() error
}
```

### 8.4. UseCase interface — `internal/indexing/interface.go`

```go
type UseCase interface {
    Index(ctx, IndexInput) (IndexOutput, error)                        // Existing (Layer 3)
    IndexInsight(ctx, IndexInsightInput) (IndexInsightOutput, error)   // NEW (Layer 2)
    IndexDigest(ctx, IndexDigestInput) (IndexDigestOutput, error)      // NEW (Layer 1)
    // ... existing retry/reconcile methods
}
```

### 8.5. Point domain — Collection routing

Point domain hiện dùng 1 collection cố định. Cần mở rộng:

- Layer 3: upsert vào `proj_{project_id}` (existing behavior)
- Layer 1 & 2: upsert vào `macro_insights` (new)

Option: truyền collection name qua `UpsertInput` hoặc tạo method riêng.

---

## 9. Migration Steps — Thứ tự thực hiện

### Phase 1: Foundation (Không break existing)

**Step 1.1** — Tạo Go structs cho 2 contract mới

- `InsightsPublishedMessage` trong `type.go`
- `ReportDigestMessage` trong `type.go`
- Sub-structs: `TopEntity`, `TopTopic`, `TopIssue`, `SeverityMix`

**Step 1.2** — Tạo Qdrant collection `macro_insights`

- Schema definition với metadata indexes
- Init script hoặc auto-create on first use

**Step 1.3** — Mở rộng Point domain hỗ trợ multi-collection

- Thêm `CollectionName` vào `UpsertInput` (hoặc tạo method mới)
- Đảm bảo Search cũng có thể target collection khác nhau

### Phase 2: Layer 2 — Insight Cards (implement trước vì đơn giản nhất)

**Step 2.1** — Thêm `IndexInsightInput`/`Output` types vào `internal/indexing/types.go`

**Step 2.2** — Implement `IndexInsight()` usecase method:

1. Validate `should_index` + `contract_version`
2. Concat `title + ". " + summary`
3. Embed via embedding domain
4. Build metadata payload
5. Generate point ID: `insight:{run_id}:{insight_type}:{hash}`
6. Upsert vào `macro_insights`

**Step 2.3** — Implement Kafka consumer `ConsumeInsightsPublished`:

- Handler: unmarshal → validate → call `IndexInsight`
- Presenter: `toIndexInsightInput()` mapping

**Step 2.4** — DLQ publisher cho `analytics.insights.published.dlq`

### Phase 3: Layer 1 — Report Digest

**Step 3.1** — Thêm `IndexDigestInput`/`Output` types

**Step 3.2** — Implement `IndexDigest()` usecase method:

1. Validate `should_index` + `contract_version`
2. Build prose text từ `top_entities`, `top_topics`, `top_issues`
3. Embed prose
4. Build metadata payload (full structured arrays vào metadata)
5. Generate point ID: `digest:{run_id}`
6. Upsert vào `macro_insights`

**Step 3.3** — Implement Kafka consumer `ConsumeReportDigest`

**Step 3.4** — DLQ publisher cho `analytics.report.digest.dlq`

### Phase 4: Layer 3 — Migrate InsightMessage struct

**Step 4.1** — Tạo `InsightMessage` nested struct trong `types.go`

- Match 1:1 với analysis-srv contract
- Giữ `AnalyticsPost` nhưng deprecated

**Step 4.2** — Update `parseJSONL()` để parse sang `InsightMessage`

**Step 4.3** — Update `indexSingleRecord()`:

- Gate: check `rag.index.should_index` thay vì `IsSpam`/`IsBot`
- Embed: dùng `content.clean_text` thay vì `Content`
- Point ID: dùng `identity.doc_id`
- Metadata: map nested fields

**Step 4.4** — Update `prepareQdrantPayload()` cho cấu trúc mới

### Phase 5: Integration & Wire-up

**Step 5.1** — Update `internal/consumer/handler.go`:

- Wire 2 consumer mới vào `domainConsumers`
- Start cả 3 consumers trong `startConsumers()`
- Graceful shutdown cho cả 3

**Step 5.2** — Update Qdrant metadata indexes

**Step 5.3** — Update reAct agent search routing (nếu trong scope)

---

## 10. Tóm tắt thay đổi theo file

| File | Thay đổi |
|------|----------|
| `internal/indexing/delivery/kafka/type.go` | Thêm topics, groups, `InsightsPublishedMessage`, `ReportDigestMessage` |
| `internal/indexing/types.go` | Thêm `InsightMessage` (nested), `IndexInsightInput/Output`, `IndexDigestInput/Output` |
| `internal/indexing/delivery/kafka/consumer/new.go` | Mở rộng `Consumer` interface, thêm consumer groups |
| `internal/indexing/delivery/kafka/consumer/consumer.go` | Thêm `ConsumeInsightsPublished()`, `ConsumeReportDigest()` |
| `internal/indexing/delivery/kafka/consumer/handler.go` | Thêm 2 handler structs mới |
| `internal/indexing/delivery/kafka/consumer/workers.go` | Thêm `handleInsightsPublished()`, `handleReportDigest()` |
| `internal/indexing/delivery/kafka/consumer/presenters.go` | Thêm `toIndexInsightInput()`, `toIndexDigestInput()` |
| `internal/indexing/usecase/index.go` | Thêm `IndexInsight()`, `IndexDigest()`, `buildDigestProse()` |
| `internal/indexing/usecase/new.go` | Không thay đổi (inject same deps) |
| `internal/indexing/interface.go` | Mở rộng `UseCase` interface |
| `internal/point/types.go` | Thêm `CollectionName` vào `UpsertInput` (hoặc new method) |
| `internal/consumer/handler.go` | Wire 2 consumer mới, start/stop |

---

## 11. Validation Checklist

Sau khi implement xong, verify:

- [ ] Consume `analytics.insights.published` → point xuất hiện trong `macro_insights` với `rag_document_type=insight_card`
- [ ] Consume `analytics.report.digest` → point xuất hiện trong `macro_insights` với `rag_document_type=report_digest`
- [ ] Consume `analytics.batch.completed` → points xuất hiện trong `proj_{project_id}` với nested metadata
- [ ] `should_index=false` → message bị skip, không có point mới
- [ ] Qdrant upsert fail → retry 3× → DLQ message published
- [ ] Duplicate `run_id` → upsert thay thế point cũ (không duplicate)
- [ ] reAct agent query tổng quan → hit Layer 1+2 từ `macro_insights`
- [ ] reAct agent query trích dẫn → hit Layer 3 từ `proj_{project_id}`
