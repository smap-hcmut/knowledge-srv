# Knowledge-srv ← Analysis-srv: Expected Contracts

> **Audience:** Analysis-srv team
> **Owner:** Knowledge-srv team
> **Date:** 2026-03-26
> **Version:** 1.1
> **Status:** Approved

---

## Tổng quan

Knowledge-srv tiêu thụ **3 Kafka topics** từ analysis-srv. Mỗi topic phục vụ 1 layer trong kiến trúc 3-layer:

| Topic | Layer | Pattern | Frequency |
|-------|-------|---------|-----------|
| `analytics.batch.completed` | Layer 3 — Per-document Evidence | Direct payload (envelope + `documents[]` array) | 1 message / run |
| `analytics.insights.published` | Layer 2 — Narrative Insights | Direct payload | 1 message / insight card (5–15 per run) |
| `analytics.report.digest` | Layer 1 — Campaign Overview | Direct payload | 1 message / run |

**Publish order obligation:** Analysis-srv PHẢI publish theo thứ tự:

```text
1. analytics.batch.completed     (Layer 3 — toàn bộ documents[] đính kèm trong message)
2. analytics.insights.published  (Layer 2 — 5–15 messages)
3. analytics.report.digest       (Layer 1 — publish CUỐI CÙNG, knowledge-srv dùng nó làm trigger export)
```

Knowledge-srv dùng `analytics.report.digest` làm tín hiệu "run hoàn thành" để trigger NotebookLM export. Nếu digest đến trước batch/insights, export sẽ thiếu data vì knowledge-srv scroll Qdrant ngay tại thời điểm nhận digest.

---

## 1. `analytics.batch.completed` — Batch Envelope (Direct Array)

**Consumer group:** `knowledge-indexing-batch`

Kafka message chứa **đầy đủ payload** — envelope metadata và toàn bộ `documents[]` là mảng các `InsightMessage` object. Knowledge-srv parse trực tiếp, không cần tải file từ ngoài.

> **Lưu ý cấu hình Kafka:** Với 2000 documents × ~2KB/document, payload ~4MB. Broker và producer PHẢI configure `max.message.bytes` ≥ 10MB (recommend 16MB).

### Kafka Message Payload

```json
{
  "project_id":  "proj_cleanser_01",
  "campaign_id": "camp_q1_2026",
  "documents": [
    {
      "identity": { "uap_id": "tt_p_fc_0001", "uap_type": "post", "uap_media_type": "video", "platform": "TIKTOK", "published_at": "2026-02-01T10:17:00Z" },
      "content":  { "clean_text": "cetaphil oily dùng da nhạy cảm thấy ổn áp ghê...", "summary": "User khen Cetaphil" },
      "nlp":      { "sentiment": { "label": "POSITIVE", "score": 0.72 }, "aspects": [{ "aspect": "GENTLE", "polarity": "POSITIVE" }], "entities": [{ "type": "BRAND", "value": "Cetaphil" }] },
      "business": { "impact": { "engagement": { "likes": 2837, "comments": 13, "shares": 27, "views": 25512 }, "impact_score": 72.5, "priority": "HIGH" } },
      "rag":      true
    },
    {
      "identity": { "uap_id": "tt_c_fc_0001_02", "uap_type": "comment", "uap_media_type": "text", "platform": "TIKTOK", "published_at": "2026-02-01T10:28:00Z" },
      "content":  { "clean_text": "ceta oily trên da mình bị rít da luôn super gentle", "summary": "User phàn nàn Cetaphil bị rít" },
      "nlp":      { "sentiment": { "label": "NEGATIVE", "score": -0.45 }, "aspects": [{ "aspect": "TEXTURE", "polarity": "NEGATIVE" }], "entities": [{ "type": "PRODUCT", "value": "Cetaphil Oily Skin Cleanser" }] },
      "business": { "impact": { "engagement": { "likes": 88, "comments": 0, "shares": 0, "views": 0 }, "impact_score": 15.3, "priority": "LOW" } },
      "rag":      true
    }
  ]
}
```

### Field Reference

| Field | Type | Required | Description | Example | Constraints |
|-------|------|----------|-------------|---------|-------------|
| `project_id` | `string` | ✅ Yes | Scope chính. Knowledge-srv tạo Qdrant collection `proj_{project_id}` để lưu documents của project này. Nếu collection chưa có, knowledge-srv tự tạo. | `"proj_cleanser_01"` | Non-empty. |
| `campaign_id` | `string` | ✅ Yes | ID campaign trong project. Lưu vào payload của mỗi Qdrant point để filter theo campaign sau này. | `"camp_q1_2026"` | Non-empty. |
| `documents` | `InsightMessage[]` | ✅ Yes | Mảng các `InsightMessage` object. Mỗi element = 1 document (POST, COMMENT, hoặc REPLY). Schema chi tiết xem Section 2. Thứ tự trong mảng không ảnh hưởng kết quả. | `[{...}, {...}]` | Non-null. Non-empty. |

---

## 2. `InsightMessage` — Per-document Record trong `documents[]`

Đây là schema cho từng element trong mảng `documents[]` của `BatchCompletedMessage`. **KHÔNG phải Kafka top-level message** — đây là nested object bên trong. Mỗi `InsightMessage` đại diện cho 1 document: POST, COMMENT, hoặc REPLY.

### Full Example — POST

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
    "sentiment": {
      "label": "POSITIVE",
      "score": 0.72
    },
    "aspects": [
      {
        "aspect": "GENTLE",
        "polarity": "POSITIVE"
      }
    ],
    "entities": [
      {
        "type": "BRAND",
        "value": "Cetaphil"
      }
    ]
  },
  "business": {
    "impact": {
      "engagement": {
        "likes": 2837,
        "comments": 13,
        "shares": 27,
        "views": 25512
      },
      "impact_score": 72.5,
      "priority": "HIGH"
    }
  },
  "rag": true
}
```

### Full Example — COMMENT

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
    "sentiment": {
      "label": "NEGATIVE",
      "score": -0.45
    },
    "aspects": [
      {
        "aspect": "TEXTURE",
        "polarity": "NEGATIVE"
      }
    ],
    "entities": [
      {
        "type": "PRODUCT",
        "value": "Cetaphil Oily Skin Cleanser"
      }
    ]
  },
  "business": {
    "impact": {
      "engagement": {
        "likes": 88,
        "comments": 0,
        "shares": 0,
        "views": 0
      },
      "impact_score": 15.3,
      "priority": "LOW"
    }
  },
  "rag": true
}
```

### Field Reference — Top-level

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `identity` | `object` | ✅ Yes | Document identity. Primary key. Xem bảng dưới. |
| `content` | `object` | ✅ Yes | Text content của document. Xem bảng dưới. |
| `nlp` | `object` | ✅ Yes | NLP enrichment results. Xem bảng dưới. |
| `business` | `object` | ✅ Yes | Business signals (engagement, impact, priority). |
| `rag` | `bool` | ✅ Yes | **Gate chính.** Knowledge-srv CHỈ index document này nếu `true`. Nếu `false`, document bị skip hoàn toàn (không Qdrant point, không error, Kafka offset vẫn được commit). Document có `rag=false` cũng KHÔNG xuất hiện trong NotebookLM export. |

### Field Reference — `identity`

| Field | Type | Required | Description | Example | Constraints |
|-------|------|----------|-------------|---------|-------------|
| `identity.uap_id` | `string` | ✅ Yes | **Primary key**. Unique per document. Dùng làm Qdrant Point ID (lưu trực tiếp dưới dạng string ID). Cross-referenced bởi Layer 2 `evidence_references[]`. Nếu empty, document bị skip ngay cả khi `rag=true`. | `"tt_p_fc_0001"` | Non-empty khi `rag=true`. Unique across toàn bộ project. |
| `identity.uap_type` | `string` | ✅ Yes | Loại document. Lưu vào Qdrant payload để filter. | `"post"` | Enum: `"post"`, `"comment"`, `"reply"` |
| `identity.uap_media_type` | `string` | ✅ Yes | Loại media của document. Lưu vào Qdrant payload để filter theo media format. | `"video"` | Enum: `"video"`, `"image"`, `"carousel"`, `"text"`, `"live"`, `"other"` |
| `identity.platform` | `string` | ✅ Yes | Platform nguồn. Lưu vào Qdrant payload. Dùng để filter và hiển thị trong search results. | `"TIKTOK"` | Enum: `"TIKTOK"`, `"FACEBOOK"`, `"INSTAGRAM"`, `"YOUTUBE"`, `"OTHER"` |
| `identity.published_at` | `string` | ✅ Yes | Thời điểm publish gốc. Lưu vào Qdrant payload cho time-range filter. | `"2026-02-01T10:17:00Z"` | RFC3339 UTC. |

### Field Reference — `content`

| Field | Type | Required | Description | Example | Ghi chú |
|-------|------|----------|-------------|---------|---------|
| `content.clean_text` | `string` | ✅ Yes | **Text duy nhất dùng để sinh embedding vector.** Text đã clean: bỏ hashtag, emoji, normalize whitespace. Knowledge-srv gọi Voyage AI embed field này, kết quả vector được lưu vào Qdrant. Nếu empty, document bị skip (STATUS_SKIPPED) ngay cả khi `rag=true`. | `"cetaphil dùng da nhạy cảm thấy ổn"` | Non-empty khi `rag=true`. |
| `content.summary` | `string` | ✅ Yes | Snippet ngắn 1 câu mô tả document. Dùng làm **display snippet** khi trả về kết quả search (field `content_summary` trong response). **KHÔNG dùng để embed.** | `"User khen Cetaphil cho da nhạy cảm"` | Recommended: < 150 chars. |

### Field Reference — `nlp`

| Field | Type | Required | Description | Example | Constraints |
|-------|------|----------|-------------|---------|-------------|
| `nlp.sentiment.label` | `string` | ✅ Yes | Sentiment label tổng thể của document. Lưu vào Qdrant payload; là filter chính khi user search theo cảm xúc. | `"POSITIVE"` | Enum: `"POSITIVE"`, `"NEGATIVE"`, `"NEUTRAL"`, `"MIXED"` |
| `nlp.sentiment.score` | `float` | ✅ Yes | Sentiment score định lượng. Lưu vào Qdrant payload. | `0.72` | Range `-1.0` (very negative) to `1.0` (very positive). |
| `nlp.aspects[]` | `array` | ✅ Yes | Danh sách aspect mentions. Mỗi element là 1 aspect được nhắc đến trong document. Empty array được chấp nhận. | `[{"aspect":"GENTLE","polarity":"POSITIVE"}]` | |
| `nlp.aspects[].aspect` | `string` | ✅ Yes | Tên aspect. Lưu vào Qdrant payload cho facet filter và aggregation. | `"GENTLE"` | Uppercase. Domain-specific vocab. |
| `nlp.aspects[].polarity` | `string` | ✅ Yes | Sentiment về aspect này cụ thể. | `"POSITIVE"` | Enum: `"POSITIVE"`, `"NEGATIVE"`, `"NEUTRAL"` |
| `nlp.entities[]` | `array` | ✅ Yes | Danh sách entities được nhắc đến trong document. Empty array được chấp nhận. | `[{"type":"BRAND","value":"Cetaphil"}]` | |
| `nlp.entities[].type` | `string` | ✅ Yes | Loại entity. | `"BRAND"` | Enum: `"BRAND"`, `"PRODUCT"`, `"PERSON"`, `"LOCATION"`, `"OTHER"` |
| `nlp.entities[].value` | `string` | ✅ Yes | Tên entity đã chuẩn hóa. Dùng để filter theo entity và cross-reference với Layer 1 `top_entities[].canonical_entity_id`. | `"Cetaphil"` | |

### Field Reference — `business`

| Field | Type | Required | Description | Example | Constraints |
|-------|------|----------|-------------|---------|-------------|
| `business.impact.engagement.likes` | `int` | ✅ Yes | Số likes. Lưu vào Qdrant payload. Hiển thị trong NotebookLM export và search results. | `2837` | ≥ 0. |
| `business.impact.engagement.comments` | `int` | ✅ Yes | Số comments. | `13` | ≥ 0. |
| `business.impact.engagement.shares` | `int` | ✅ Yes | Số shares/reposts. | `27` | ≥ 0. |
| `business.impact.engagement.views` | `int` | ✅ Yes | Số views. | `25512` | ≥ 0. |
| `business.impact.impact_score` | `float` | ✅ Yes | Composite importance score. Knowledge-srv dùng ngưỡng **`impact_score > 60`** là một trong hai điều kiện OR để chọn top documents đưa vào NotebookLM export. Document xuất hiện trong export nếu `priority = "HIGH"` **hoặc** `impact_score > 60`. | `72.5` | Range `0.0–100.0`. |
| `business.impact.priority` | `string` | ✅ Yes | Priority tier. Knowledge-srv dùng **`priority = "HIGH"`** là điều kiện OR còn lại cho NotebookLM export (xem mô tả `impact_score`). Ngoài ra còn là filter khi user search theo priority. | `"HIGH"` | Enum: `"LOW"`, `"MEDIUM"`, `"HIGH"`, `"CRITICAL"` |

---

## 3. `analytics.insights.published` — Insight Card

**Consumer group:** `knowledge-indexing-insights`

Mỗi insight card = 1 Kafka message riêng. Mỗi run publish 5–15 messages, hiện tại thực tế là 7 (1 per `insight_type`). Tất cả messages cùng run phải có cùng `run_id`.

### Full Example

```json
{
  "project_id":   "proj_cleanser_01",
  "campaign_id":  "camp_q1_2026",
  "run_id":       "run-20260323T165146Z",

  "insight_type": "share_of_voice_shift",
  "title":        "Share of voice shift detected for Cetaphil",
  "summary":      "Cetaphil lost 22 mention(s) in the later half of the window.",
  "confidence":   0.5612,

  "analysis_window_start": "2026-02-01T10:17:00Z",
  "analysis_window_end":   "2026-02-10T21:24:00Z",

  "supporting_metrics": {
    "mention_share":      0.117,
    "delta_mention_share": -0.0112
  },
  "evidence_references": [
    "tt_p_fc_0001",
    "tt_c_fc_0001_02",
    "tt_c_fc_0001_03",
    "tt_c_fc_0001_06",
    "tt_c_fc_0001_09"
  ],
  "should_index": true
}
```

### Field Reference

| Field | Type | Required | Description | Example | Constraints |
|-------|------|----------|-------------|---------|-------------|
| `project_id` | `string` | ✅ Yes | Scope filter. Match với `project_id` trong Layer 3. Lưu vào Qdrant payload của insight point. | `"proj_cleanser_01"` | Non-empty. |
| `campaign_id` | `string` | ✅ Yes | Campaign ID. Lưu vào Qdrant payload. | `"camp_q1_2026"` | Non-empty. |
| `run_id` | `string` | ✅ Yes | ID của analysis run. Dùng làm phần của Qdrant Point ID (`insight:{run_id}:{insight_type}:{hash}`). **PHẢI consistent với `run_id` trong `analytics.report.digest` cùng run** — knowledge-srv dùng `run_id` để scroll `macro_insights` collection khi build NotebookLM export. | `"run-20260323T165146Z"` | Non-empty. Format: `run-{YYYYMMDD}T{HHMMSS}Z`. |
| `insight_type` | `string` | ✅ Yes | Loại insight. Cùng với `run_id`, tạo thành dedup key khi upsert vào Qdrant. Xem enum bên dưới. | `"share_of_voice_shift"` | Enum (xem bảng insight_type). |
| `title` | `string` | ✅ Yes | Tiêu đề insight ngắn gọn. **Phần 1 của embed text** — knowledge-srv embed chuỗi `"{title}. {summary}"`. Nếu empty, message bị skip. | `"Share of voice shift detected for Cetaphil"` | Non-empty. |
| `summary` | `string` | ✅ Yes | Mô tả chi tiết insight. **Phần 2 của embed text** (quan trọng nhất về ngữ nghĩa). Knowledge-srv embed `"{title}. {summary}"`. Ngoài ra còn hiển thị verbatim trong NotebookLM export. Nếu empty, message bị skip. | `"Cetaphil lost 22 mention(s)..."` | Non-empty. |
| `confidence` | `float` | ✅ Yes | Độ tin cậy của insight. Lưu vào Qdrant payload. Dùng cho threshold filter khi query. | `0.5612` | Range `0.0–1.0`. |
| `analysis_window_start` | `string` | ✅ Yes | Bắt đầu window phân tích. Lưu vào Qdrant payload. | `"2026-02-01T10:17:00Z"` | RFC3339 UTC. |
| `analysis_window_end` | `string` | ✅ Yes | Kết thúc window phân tích. Lưu vào Qdrant payload. | `"2026-02-10T21:24:00Z"` | RFC3339 UTC. |
| `supporting_metrics` | `object` | ✅ Yes | Key-value object linh hoạt chứa số liệu hỗ trợ. Lưu nguyên vào Qdrant payload. Schema cố định theo `insight_type` (xem bảng dưới). | `{"mention_share": 0.117}` | JSON object. Không được null. |
| `evidence_references` | `string[]` | ✅ Yes | Mảng `uap_id` của các documents (Layer 3) làm bằng chứng cho insight này. Lưu vào Qdrant payload; hiển thị trong NotebookLM export. Mỗi `uap_id` nên có trong `documents[]` của `analytics.batch.completed` cùng run. | `["tt_p_fc_0001", "tt_c_fc_0001_02"]` | Non-null. Empty array được chấp nhận. |
| `should_index` | `bool` | ✅ Yes | Gate chính. `false` → knowledge-srv skip toàn bộ (không Qdrant, không error, offset commit). | `true` | |

### `insight_type` Enum

| Value | Sinh từ BI report | Ý nghĩa |
|-------|------------------|---------|
| `share_of_voice_shift` | `sov_report` | Brand tăng/giảm thị phần mention trong window |
| `trending_topic` | `buzz_report` (topic section) | Topic đang tăng volume nhanh |
| `conversation_hotspot` | `buzz_report` (topic section) | Topic có buzz score / engagement cao |
| `emerging_topic` | `emerging_topics_report` | Topic mới xuất hiện lần đầu trong window này |
| `issue_warning` | `top_issues_report` | Issue category có pressure score cao |
| `controversy_alert` | `thread_controversy_report` | Thread có tranh luận bất thường |
| `creator_concentration` | `creator_source_breakdown_report` | Tập trung mentions từ ít tác giả (signal bất thường) |

### `supporting_metrics` per `insight_type`

Các key trong `supporting_metrics` là bắt buộc theo chuẩn sau:

| `insight_type` | Expected keys trong `supporting_metrics` |
|---|---|
| `share_of_voice_shift` | `mention_share`, `delta_mention_share` |
| `trending_topic` | `mention_count`, `growth_rate`, `buzz_score_proxy` |
| `conversation_hotspot` | `buzz_score_proxy`, `mention_count` |
| `emerging_topic` | `mention_count`, `quality_score` |
| `issue_warning` | `mention_count`, `issue_pressure_proxy`, `severity_mix` |
| `controversy_alert` | `controversy_score`, `thread_id` |
| `creator_concentration` | `top_creator_mention_share`, `author_id` |

---

## 4. `analytics.report.digest` — Run Digest

**Consumer group:** `knowledge-indexing-digest`

1 message duy nhất per run. Đây là message **CUỐI CÙNG** được publish trong run. Knowledge-srv dùng nó làm trigger cho NotebookLM export: sau khi upsert digest point vào Qdrant, một goroutine background được khởi chạy để đọc lại toàn bộ Layer 1 + 2 + 3 và upload lên NotebookLM qua Maestro.

### Full Example

```json
{
  "project_id":   "proj_cleanser_01",
  "campaign_id":  "camp_q1_2026",
  "run_id":       "run-20260323T165146Z",

  "analysis_window_start": "2026-02-01T10:17:00Z",
  "analysis_window_end":   "2026-02-10T21:24:00Z",

  "domain_overlay":  "domain-facial-cleanser-vn",
  "platform":        "tiktok",
  "total_mentions":  2000,

  "top_entities": [
    {
      "canonical_entity_id": "brand.cetaphil",
      "entity_name":         "Cetaphil",
      "entity_type":         "brand",
      "mention_count":       234,
      "mention_share":       0.117
    },
    {
      "canonical_entity_id": "brand.cerave",
      "entity_name":         "CeraVe",
      "entity_type":         "brand",
      "mention_count":       233,
      "mention_share":       0.1165
    }
  ],

  "top_topics": [
    {
      "topic_key":          "cleanser_brand_comparison",
      "topic_label":        "Cleanser Brand and Product Comparison",
      "mention_count":      364,
      "mention_share":      0.182,
      "buzz_score_proxy":   429.37,
      "quality_score":      0.9752,
      "representative_texts": ["Dùng thử 5 loại sữa rửa mặt...", "So sánh cetaphil vs cerave..."]
    },
    {
      "topic_key":          "cleanser_recommendation_by_skin_type",
      "topic_label":        "Cleanser Recommendation by Skin Type",
      "mention_count":      375,
      "mention_share":      0.1875,
      "buzz_score_proxy":   null,
      "quality_score":      0.9752,
      "representative_texts": ["Da dầu nên dùng gì?", "Cho da nhạy cảm recommend..."]
    }
  ],

  "top_issues": [
    {
      "issue_category":            "fake_authenticity_concern",
      "mention_count":             200,
      "issue_pressure_proxy":      149.477,
      "severity_mix": {
        "low":    0.6,
        "medium": 0.3,
        "high":   0.1
      }
    }
  ],

  "should_index": true
}
```

### Field Reference — Top-level

| Field | Type | Required | Description | Constraints |
|-------|------|----------|-------------|-------------|
| `project_id` | `string` | ✅ Yes | Project scope. Dùng để scroll Qdrant collection `proj_{project_id}` khi build NotebookLM export (Layer 3 documents). | Non-empty. |
| `campaign_id` | `string` | ✅ Yes | Campaign ID. Lưu vào Qdrant payload. **Bắt buộc non-empty để trigger NotebookLM sync** — knowledge-srv kiểm tra `campaign_id != ""` trước khi khởi goroutine export. Nếu empty, sync bị skip silently. | Non-empty. |
| `run_id` | `string` | ✅ Yes | Run ID. Lưu vào Qdrant point ID (`digest:{run_id}`). **PHẢI consistent với `run_id` trong tất cả Layer 2 messages cùng run** — knowledge-srv dùng `run_id` để scroll `macro_insights` collection (insight cards + digest). | Non-empty. Format: `run-{YYYYMMDD}T{HHMMSS}Z`. |
| `analysis_window_start` | `string` | ✅ Yes | Bắt đầu window phân tích. Lưu vào Qdrant payload. **Cũng được knowledge-srv dùng để tính ISO week label** cho tiêu đề NotebookLM part (e.g. `SMAP | camp_q1_2026 | 2026-W09`). | RFC3339 UTC. |
| `analysis_window_end` | `string` | ✅ Yes | Kết thúc window phân tích. Lưu vào Qdrant payload. | RFC3339 UTC. |
| `domain_overlay` | `string` | ✅ Yes | Domain slug. Dùng để build prose trong Qdrant embed text và làm header section trong NotebookLM export: `Campaign Report: {domain_overlay}`. | Non-empty. |
| `platform` | `string` | ✅ Yes | Platform chính của batch. Lưu vào Qdrant payload. Xuất hiện trong NotebookLM export header. | Enum: `"tiktok"`, `"facebook"`, `"instagram"`, `"youtube"`, `"multi"` |
| `total_mentions` | `int` | ✅ Yes | Tổng số documents trong batch (posts + comments + replies). Lưu vào Qdrant payload. Hiển thị trong NotebookLM export: `Total Mentions: {total_mentions}`. | > 0. |
| `top_entities` | `array` | ✅ Yes | Top entities sorted by `mention_count` desc. Xuất hiện trong section "Top Brands" của NotebookLM export (lấy top 5). Recommended: top 10. | Non-null. |
| `top_topics` | `array` | ✅ Yes | Top topics sorted by `mention_count` desc. Xuất hiện trong section "Key Discussion Topics" của NotebookLM export (lấy top 5). Recommended: top 10. | Non-null. |
| `top_issues` | `array` | ✅ Yes | Top issues sorted by `issue_pressure_proxy` desc. Xuất hiện trong section "Critical Issues" của NotebookLM export (lấy top 5). Recommended: top 10. | Non-null. |
| `should_index` | `bool` | ✅ Yes | Gate. `false` → skip cả Qdrant indexing VÀ NotebookLM export trigger. Kafka offset vẫn commit. | |

### Field Reference — `top_entities[]`

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `canonical_entity_id` | `string` | ✅ Yes | Canonical ID. Lưu vào Qdrant payload. Dùng để cross-reference với Layer 2 insight cards khi cần drill-down. | `"brand.cetaphil"` |
| `entity_name` | `string` | ✅ Yes | Tên hiển thị chuẩn hóa. Xuất hiện verbatim trong NotebookLM export: `- {entity_name}: {mention_count} mentions ({mention_share*100}% share)`. | `"Cetaphil"` |
| `entity_type` | `string` | ✅ Yes | Loại entity. Lưu vào Qdrant payload. | Enum: `"brand"`, `"product"`, `"person"`, `"topic"` |
| `mention_count` | `int` | ✅ Yes | Số lần mention trong toàn bộ batch (posts + comments + replies). | `234` |
| `mention_share` | `float` | ✅ Yes | `mention_count / total_mentions`. | Range `0.0–1.0`. |

### Field Reference — `top_topics[]`

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `topic_key` | `string` | ✅ Yes | Slug key của topic. Unique per domain. Lưu vào Qdrant payload. | `"cleanser_brand_comparison"` |
| `topic_label` | `string` | ✅ Yes | Tên hiển thị đầy đủ. Xuất hiện verbatim trong NotebookLM export: `- {topic_label}: {mention_count} mentions`. | `"Cleanser Brand and Product Comparison"` |
| `mention_count` | `int` | ✅ Yes | Số documents liên quan đến topic này. | `364` |
| `mention_share` | `float` | ✅ Yes | `mention_count / total_mentions`. | Range `0.0–1.0`. |
| `buzz_score_proxy` | `float` | Optional | Composite engagement-weighted buzz score. Null nếu không tính được — knowledge-srv sẽ omit khỏi NotebookLM export nếu null. | `429.37` |
| `quality_score` | `float` | Optional | Topic quality score (coverage + coherence). Xuất hiện trong NotebookLM export nếu non-null. Null nếu không tính được. | `0.9752` |
| `representative_texts` | `string[]` | Optional | 1–3 sample texts đại diện cho topic. Xuất hiện trong NotebookLM export phần "Example": `Example: "{representative_texts[0]}"`. Null/empty → omit. | `["sample text 1"]` |

### Field Reference — `top_issues[]`

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `issue_category` | `string` | ✅ Yes | Issue category key. Xuất hiện verbatim trong NotebookLM export: `- {issue_category}: {mention_count} mentions (pressure: {issue_pressure_proxy})`. | `"fake_authenticity_concern"` |
| `mention_count` | `int` | ✅ Yes | Số documents có issue này. | `200` |
| `issue_pressure_proxy` | `float` | ✅ Yes | Composite pressure score (mentions × severity weights). Dùng để sort và xuất hiện trong NotebookLM export. | `149.477` |
| `severity_mix` | `object` | Optional | Phân bổ severity. Null nếu không tính được — knowledge-srv sẽ omit nếu null. | `{"low": 0.6, "medium": 0.3, "high": 0.1}` |
| `severity_mix.low` | `float` | Optional | Tỷ lệ documents ở severity LOW. | `0.6` |
| `severity_mix.medium` | `float` | Optional | Tỷ lệ documents ở severity MEDIUM. | `0.3` |
| `severity_mix.high` | `float` | Optional | Tỷ lệ documents ở severity HIGH. | `0.1` |

---

## 5. Conventions & Shared Rules

### `run_id` Consistency

Cùng 1 analysis run PHẢI dùng cùng `run_id` trong:

- Tất cả Layer 2 insight card messages (`analytics.insights.published`)
- Layer 1 digest message (`analytics.report.digest`)

Format khuyến nghị: `run-{YYYYMMDD}T{HHMMSS}Z` (compact ISO 8601, UTC).

Layer 3 (`analytics.batch.completed`) KHÔNG có `run_id` — chỉ có `project_id` và `campaign_id`. Knowledge-srv không link Layer 3 với `run_id`.

### `should_index` Gate

`should_index = false` có nghĩa:

- Analysis-srv đã tính toán xong nhưng quyết định document/insight/digest này KHÔNG nên index
- Knowledge-srv KHÔNG tạo Qdrant point
- Knowledge-srv KHÔNG publish DLQ
- Knowledge-srv vẫn commit Kafka offset (không retry)

Khi nào `should_index = false`? Ví dụ:

- Document là spam nhưng vẫn có trong `documents[]`
- Insight có confidence quá thấp (< 0.3)
- Run là test run, không phải production data

### Timestamp Format

Tất cả timestamps phải là RFC3339 UTC: `YYYY-MM-DDTHH:MM:SSZ`. Không dùng timezone offset.

### Kafka Message Size Limits

- `analytics.batch.completed`: **lớn (~4MB)** — direct payload với 2000 documents. Broker, producer và consumer PHẢI configure `max.message.bytes` ≥ 10MB (recommend 16MB).
- `analytics.insights.published`: nhỏ (< 2KB per message)
- `analytics.report.digest`: trung bình (< 20KB — tùy top_entities/topics/issues arrays)

---

## 6. Điều knowledge-srv KHÔNG cần từ analysis-srv

Để tránh confusion:

| Item | Lý do KHÔNG cần |
|------|-----------------|
| `smap.analytics.output` topic | Legacy topic, chưa bao giờ implement. **Không tạo producer cho topic này.** |
| Pre-computed embedding vectors | Knowledge-srv tự embed qua Voyage AI — analysis-srv không cần cung cấp vectors. |
| Qdrant collection management | Knowledge-srv tự tạo/manage collections (`proj_{project_id}`, `macro_insights`). |
| Full comment hierarchy nested trong POST | Mỗi comment/reply là 1 InsightMessage riêng với `identity.uap_type = "comment"/"reply"`. Không nest vào parent. |
