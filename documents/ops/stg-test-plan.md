# Staging Test Plan — Knowledge-srv NotebookLM Integration

> **Environment:** STG
> **Date:** 2026-03-26
> **Owner:** Knowledge-srv team
> **Prerequisite:** Maestro service is live and accessible from STG

---

## Mục tiêu

Xác minh end-to-end flow từ Kafka message → Qdrant indexing → NotebookLM sync (qua Maestro) → Chat API hoạt động đúng trên môi trường STG trước khi ship PROD.

---

## T0 — Kiểm tra hạ tầng trước khi test

### T0-1: Service health

```bash
# Knowledge-srv health
curl -f https://smap-api.tantai.dev/knowledge/health

# Qdrant reachable (từ knowledge-srv pod)
curl -f http://qdrant.stg.internal:6334/health

# Kafka topics tồn tại
kafka-topics.sh --bootstrap-server kafka.stg.internal:9092 --list | grep analytics
# Expected: analytics.batch.completed, analytics.insights.published, analytics.report.digest
```

### T0-2: Config check

Xác nhận trong STG config:

```yaml
notebook:
  enabled: true
  sync_max_retries: 3
  chat_timeout_sec: 30

maestro:
  base_url: "https://maestro.stg.internal"
  webhook_callback_url: "https://smap-api.tantai.dev/knowledge/webhook/maestro"
  job_poll_interval_ms: 2000
  job_poll_max_attempts: 15

router:
  notebook_fallback_enabled: true
```

### T0-3: Maestro connectivity

```bash
# Verify maestro endpoint (thực hiện sau khi đảm bảo Maestro live)
curl -f https://maestro.stg.internal/health
```

**Nếu T0 fail → STOP, không tiếp tục.**

---

## T1 — Layer 3: `analytics.batch.completed`

### Mục tiêu

Documents được index vào Qdrant collection `proj_{project_id}`.

### T1-1: Publish test message

```bash
kafka-console-producer.sh \
  --bootstrap-server kafka.stg.internal:9092 \
  --topic analytics.batch.completed <<'EOF'
{
  "project_id": "proj_stg_test_01",
  "campaign_id": "camp_stg_q1",
  "documents": [
    {
      "identity": { "uap_id": "stg_post_001", "uap_type": "post", "uap_media_type": "video", "platform": "TIKTOK", "published_at": "2026-03-01T10:00:00Z" },
      "content":  { "clean_text": "cetaphil dưỡng ẩm da dầu rất tốt không bị nhờn", "summary": "User khen cetaphil dưỡng ẩm" },
      "nlp":      { "sentiment": { "label": "POSITIVE", "score": 0.80 }, "aspects": [{"aspect": "MOISTURE", "polarity": "POSITIVE"}], "entities": [{"type": "BRAND", "value": "Cetaphil"}] },
      "business": { "impact": { "engagement": { "likes": 500, "comments": 20, "shares": 15, "views": 10000 }, "impact_score": 75.0, "priority": "HIGH" } },
      "rag": true
    },
    {
      "identity": { "uap_id": "stg_post_002", "uap_type": "comment", "uap_media_type": "text", "platform": "TIKTOK", "published_at": "2026-03-01T11:00:00Z" },
      "content":  { "clean_text": "cerave foam cleanser da nhạy cảm dùng được không", "summary": "User hỏi về CeraVe cho da nhạy cảm" },
      "nlp":      { "sentiment": { "label": "NEUTRAL", "score": 0.0 }, "aspects": [], "entities": [{"type": "PRODUCT", "value": "CeraVe Foam Cleanser"}] },
      "business": { "impact": { "engagement": { "likes": 10, "comments": 2, "shares": 0, "views": 0 }, "impact_score": 12.0, "priority": "LOW" } },
      "rag": true
    },
    {
      "identity": { "uap_id": "stg_post_003", "uap_type": "post", "uap_media_type": "image", "platform": "INSTAGRAM", "published_at": "2026-03-01T12:00:00Z" },
      "content":  { "clean_text": "spam content không liên quan", "summary": "Spam" },
      "nlp":      { "sentiment": { "label": "NEUTRAL", "score": 0.0 }, "aspects": [], "entities": [] },
      "business": { "impact": { "engagement": { "likes": 0, "comments": 0, "shares": 0, "views": 0 }, "impact_score": 0.0, "priority": "LOW" } },
      "rag": false
    }
  ]
}
EOF
```

### T1-2: Verify Qdrant indexing

```bash
# Kiểm tra collection tồn tại
curl http://qdrant.stg.internal:6333/collections/proj_stg_test_01

# Kiểm tra số points (phải là 2, vì stg_post_003 có rag=false)
curl http://qdrant.stg.internal:6333/collections/proj_stg_test_01/points/count

# Kiểm tra stg_post_001 được index
curl -X POST http://qdrant.stg.internal:6333/collections/proj_stg_test_01/points/scroll \
  -H "Content-Type: application/json" \
  -d '{"filter": {"must": [{"key": "uap_id", "match": {"value": "stg_post_001"}}]}, "with_payload": true}'
```

**Expected:**

- Collection `proj_stg_test_01` tồn tại
- `points_count = 2` (stg_post_001 + stg_post_002)
- stg_post_003 KHÔNG có trong collection (rag=false)
- stg_post_001 payload có `priority: "HIGH"`, `impact_score: 75.0`, `campaign_id: "camp_stg_q1"`

### T1-3: Test rag=false không tạo point

Xác nhận stg_post_003 không tồn tại trong Qdrant:

```bash
curl -X POST http://qdrant.stg.internal:6333/collections/proj_stg_test_01/points/scroll \
  -H "Content-Type: application/json" \
  -d '{"filter": {"must": [{"key": "uap_id", "match": {"value": "stg_post_003"}}]}, "with_payload": false}'
# Expected: result.points = []
```

---

## T2 — Layer 2: `analytics.insights.published`

### Mục tiêu

Insight cards được index vào Qdrant collection `macro_insights`.

### T2-1: Publish test message

```bash
kafka-console-producer.sh \
  --bootstrap-server kafka.stg.internal:9092 \
  --topic analytics.insights.published <<'EOF'
{
  "project_id":   "proj_stg_test_01",
  "campaign_id":  "camp_stg_q1",
  "run_id":       "run-20260326T100000Z",
  "insight_type": "share_of_voice_shift",
  "title":        "Cetaphil gained share of voice in cleanser segment",
  "summary":      "Cetaphil tăng 15 mention(s) trong nửa sau window, vượt CeraVe về thị phần discussion.",
  "confidence":   0.78,
  "analysis_window_start": "2026-03-01T00:00:00Z",
  "analysis_window_end":   "2026-03-14T23:59:59Z",
  "supporting_metrics": { "mention_share": 0.21, "delta_mention_share": 0.05 },
  "evidence_references": ["stg_post_001", "stg_post_002"],
  "should_index": true
}
EOF
```

### T2-2: Verify insight point trong Qdrant

```bash
curl -X POST http://qdrant.stg.internal:6333/collections/macro_insights/points/scroll \
  -H "Content-Type: application/json" \
  -d '{
    "filter": {
      "must": [
        {"key": "run_id", "match": {"value": "run-20260326T100000Z"}},
        {"key": "insight_type", "match": {"value": "share_of_voice_shift"}}
      ]
    },
    "with_payload": true
  }'
```

**Expected:**

- 1 point tồn tại trong `macro_insights`
- Payload có `title`, `summary`, `confidence: 0.78`, `run_id: "run-20260326T100000Z"`
- `evidence_references` = `["stg_post_001", "stg_post_002"]`

### T2-3: Test should_index=false

```bash
kafka-console-producer.sh \
  --bootstrap-server kafka.stg.internal:9092 \
  --topic analytics.insights.published <<'EOF'
{
  "project_id": "proj_stg_test_01", "campaign_id": "camp_stg_q1",
  "run_id": "run-20260326T100000Z", "insight_type": "trending_topic",
  "title": "Test skip", "summary": "Should not be indexed",
  "confidence": 0.1, "analysis_window_start": "2026-03-01T00:00:00Z",
  "analysis_window_end": "2026-03-14T23:59:59Z",
  "supporting_metrics": {}, "evidence_references": [],
  "should_index": false
}
EOF
```

Verify `trending_topic` insight KHÔNG được index:

```bash
curl -X POST http://qdrant.stg.internal:6333/collections/macro_insights/points/scroll \
  -H "Content-Type: application/json" \
  -d '{"filter": {"must": [{"key": "insight_type", "match": {"value": "trending_topic"}}]}, "with_payload": false}'
# Expected: result.points = []
```

---

## T3 — Layer 1: `analytics.report.digest` + NotebookLM sync trigger

### Mục tiêu

Digest được index vào `macro_insights`; NotebookLM export goroutine được khởi chạy async.

### T3-1: Publish digest message

> **Quan trọng:** Publish SAU khi T1 và T2 hoàn thành để NotebookLM export có đủ data.

```bash
kafka-console-producer.sh \
  --bootstrap-server kafka.stg.internal:9092 \
  --topic analytics.report.digest <<'EOF'
{
  "project_id":   "proj_stg_test_01",
  "campaign_id":  "camp_stg_q1",
  "run_id":       "run-20260326T100000Z",
  "analysis_window_start": "2026-03-01T00:00:00Z",
  "analysis_window_end":   "2026-03-14T23:59:59Z",
  "domain_overlay":  "domain-facial-cleanser-vn",
  "platform":        "tiktok",
  "total_mentions":  500,
  "top_entities": [
    {"canonical_entity_id": "brand.cetaphil", "entity_name": "Cetaphil", "entity_type": "brand", "mention_count": 150, "mention_share": 0.30},
    {"canonical_entity_id": "brand.cerave",   "entity_name": "CeraVe",   "entity_type": "brand", "mention_count": 120, "mention_share": 0.24}
  ],
  "top_topics": [
    {"topic_key": "brand_comparison", "topic_label": "Cleanser Brand Comparison", "mention_count": 200, "mention_share": 0.40, "buzz_score_proxy": 350.0, "quality_score": 0.88, "representative_texts": ["Cetaphil vs CeraVe test trên da dầu"]}
  ],
  "top_issues": [
    {"issue_category": "counterfeit_product", "mention_count": 50, "issue_pressure_proxy": 38.5, "severity_mix": {"low": 0.5, "medium": 0.3, "high": 0.2}}
  ],
  "should_index": true
}
EOF
```

### T3-2: Verify digest point trong Qdrant

```bash
curl -X POST http://qdrant.stg.internal:6333/collections/macro_insights/points/scroll \
  -H "Content-Type: application/json" \
  -d '{
    "filter": {
      "must": [
        {"key": "rag_document_type", "match": {"value": "digest"}},
        {"key": "run_id", "match": {"value": "run-20260326T100000Z"}}
      ]
    },
    "with_payload": true
  }'
```

**Expected:** 1 point với `rag_document_type = "digest"`, payload có `top_entities`, `top_topics`, `top_issues`.

### T3-3: Verify NotebookLM sync được trigger

Kiểm tra logs của knowledge-srv ngay sau khi publish digest:

```bash
# Tìm log goroutine khởi chạy
kubectl logs -n stg deployment/knowledge-srv --since=5m | grep -E "SyncPart|EnsureNotebook|BuildParts|notebook"
```

**Expected logs (theo thứ tự, có thể cách nhau vài giây):**

```
transform.buildParts: starting for campaign=camp_stg_q1
transform.buildParts: scrolled N digest points
transform.buildParts: scrolled M insight points
transform.buildParts: scrolled K high-impact posts
transform.buildParts: built X parts, total ~Ybytes
notebook.EnsureNotebook: checking existing notebook for campaign=camp_stg_q1
notebook.UploadSources: uploading X parts to maestro
```

### T3-4: Verify notebook record trong PostgreSQL

```sql
-- Kiểm tra campaign record tạo thành công
SELECT * FROM nb_campaigns
WHERE campaign_id = 'camp_stg_q1'
ORDER BY created_at DESC
LIMIT 1;
-- Expected: 1 row với notebook_id non-empty

-- Kiểm tra source records
SELECT id, campaign_id, content_hash, status, maestro_job_id
FROM nb_sources
WHERE campaign_id = 'camp_stg_q1'
ORDER BY created_at DESC;
-- Expected: 1+ rows với status = 'uploading' hoặc 'synced' (tùy tốc độ Maestro)
```

---

## T4 — Maestro Webhook Callback

### Mục tiêu

Knowledge-srv nhận và xử lý webhook từ Maestro khi upload sources hoàn tất.

### T4-1: Simulate webhook (nếu Maestro chưa live hoặc để test riêng)

Lấy `maestro_job_id` từ `nb_sources` table (xem T3-4), sau đó:

```bash
# Simulate upload_sources_completed webhook
MAESTRO_JOB_ID="<lấy từ db>"

curl -X POST https://smap-api.tantai.dev/knowledge/webhook/maestro \
  -H "Content-Type: application/json" \
  -H "X-Maestro-Secret: <webhook_secret_from_config>" \
  -d "{
    \"event\": \"upload_sources_completed\",
    \"data\": {
      \"jobId\": \"$MAESTRO_JOB_ID\",
      \"status\": \"COMPLETED\",
      \"result\": {}
    }
  }"
```

**Expected:** HTTP 200. Verify trong DB:

```sql
SELECT status FROM nb_sources WHERE maestro_job_id = '<MAESTRO_JOB_ID>';
-- Expected: status = 'synced'
```

### T4-2: Simulate upload_sources_failed

```bash
curl -X POST https://smap-api.tantai.dev/knowledge/webhook/maestro \
  -H "Content-Type: application/json" \
  -H "X-Maestro-Secret: <webhook_secret>" \
  -d "{
    \"event\": \"upload_sources_completed\",
    \"data\": {
      \"jobId\": \"$MAESTRO_JOB_ID\",
      \"status\": \"FAILED\",
      \"result\": {}
    }
  }"
```

**Expected:** HTTP 200. Source record cập nhật `status = 'failed'`.

---

## T5 — Chat API: Qdrant flow (notebook disabled hoặc unavailable)

### Mục tiêu

Chat hoạt động bình thường qua Qdrant khi notebook chưa sync hoặc intent là structured.

### T5-1: Structured intent query (always routes to Qdrant)

```bash
curl -X POST https://smap-api.tantai.dev/knowledge/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <stg_token>" \
  -d '{
    "project_id": "proj_stg_test_01",
    "campaign_id": "camp_stg_q1",
    "message": "đếm số mentions của Cetaphil trong tuần qua",
    "conversation_id": ""
  }'
```

**Expected:** HTTP 200, body có `answer` (từ Gemini + Qdrant context), `is_async: false`.

### T5-2: Narrative query khi notebook chưa sync

```bash
curl -X POST https://smap-api.tantai.dev/knowledge/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <stg_token>" \
  -d '{
    "project_id": "proj_stg_test_01",
    "campaign_id": "camp_stg_NONEXISTENT",
    "message": "hãy phân tích xu hướng thị trường tuần này",
    "conversation_id": ""
  }'
```

**Expected:** HTTP 200, `is_async: false` (fallback to Qdrant vì không có notebook cho campaign này).

---

## T6 — Chat API: Async NotebookLM flow

> **Prerequisite:** T3-4 hoàn thành và `nb_sources.status = 'synced'` cho `camp_stg_q1`.

### Mục tiêu

Narrative query được route sang NotebookLM async job; client poll để lấy kết quả.

### T6-1: Submit narrative chat query

```bash
curl -X POST https://smap-api.tantai.dev/knowledge/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <stg_token>" \
  -d '{
    "project_id": "proj_stg_test_01",
    "campaign_id": "camp_stg_q1",
    "message": "hãy phân tích insight nổi bật và xu hướng tuần này",
    "conversation_id": ""
  }'
```

**Expected:** HTTP 202 Accepted:

```json
{
  "chat_job_id": "job_<uuid>",
  "conversation_id": "<uuid>",
  "status": "pending",
  "is_async": true
}
```

### T6-2: Poll job status

```bash
JOB_ID="<chat_job_id từ T6-1>"

# Poll ngay sau khi submit (should be processing)
curl https://smap-api.tantai.dev/knowledge/chat/jobs/$JOB_ID \
  -H "Authorization: Bearer <stg_token>"
# Expected: HTTP 202, status = "processing"

# Poll sau 30-60 giây (Maestro đã xử lý)
sleep 45
curl https://smap-api.tantai.dev/knowledge/chat/jobs/$JOB_ID \
  -H "Authorization: Bearer <stg_token>"
# Expected: HTTP 200, status = "completed", answer non-empty
```

### T6-3: Webhook cho chat job (simulate nếu cần)

```bash
MAESTRO_CHAT_JOB_ID="<lấy từ nb_chat_jobs table>"

curl -X POST https://smap-api.tantai.dev/knowledge/webhook/maestro \
  -H "Content-Type: application/json" \
  -H "X-Maestro-Secret: <webhook_secret>" \
  -d "{
    \"event\": \"chat_completed\",
    \"data\": {
      \"jobId\": \"$MAESTRO_CHAT_JOB_ID\",
      \"status\": \"COMPLETED\",
      \"result\": {
        \"answer\": \"Cetaphil tăng trưởng mạnh trong tuần, đặc biệt segment da dầu. CeraVe vẫn dẫn đầu về volume nhưng xu hướng đang chuyển dịch.\"
      }
    }
  }"
```

Poll lại job → Expected HTTP 200, `answer` có nội dung từ webhook.

### T6-4: Fallback khi Maestro timeout

Với config `chat_timeout_sec = 30` và `notebook_fallback_enabled = true`:

1. Submit chat query (T6-1)
2. **Không** simulate webhook
3. Poll sau 35 giây:

```bash
sleep 35
curl https://smap-api.tantai.dev/knowledge/chat/jobs/$JOB_ID \
  -H "Authorization: Bearer <stg_token>"
```

**Expected:** HTTP 200, `status = "completed"`, `answer` có nội dung từ Qdrant fallback (khác với NotebookLM answer nhưng non-empty).

---

## T7 — Edge Cases

### T7-1: Job ID không tồn tại

```bash
curl https://smap-api.tantai.dev/knowledge/chat/jobs/nonexistent-job-id \
  -H "Authorization: Bearer <stg_token>"
# Expected: HTTP 404
```

### T7-2: Digest với should_index=false

```bash
kafka-console-producer.sh --bootstrap-server kafka.stg.internal:9092 --topic analytics.report.digest <<'EOF'
{
  "project_id": "proj_stg_test_01", "campaign_id": "camp_stg_skip",
  "run_id": "run-skip", "analysis_window_start": "2026-03-01T00:00:00Z",
  "analysis_window_end": "2026-03-14T23:59:59Z",
  "domain_overlay": "test", "platform": "tiktok", "total_mentions": 10,
  "top_entities": [], "top_topics": [], "top_issues": [],
  "should_index": false
}
EOF
```

Verify: KHÔNG có digest point cho `run_id = "run-skip"` trong Qdrant, KHÔNG có notebook record cho `camp_stg_skip`.

### T7-3: Idempotency — gửi lại batch cùng documents

Gửi lại message T1-1 lần 2. Verify:

```bash
curl http://qdrant.stg.internal:6333/collections/proj_stg_test_01/points/count
# Expected: vẫn là 2 (không tạo thêm point, upsert idempotent)
```

---

## Checklist tổng hợp

| Test | Mô tả                      | Pass Criteria                     | Status |
| ---- | -------------------------- | --------------------------------- | ------ |
| T0-1 | Service health             | All healthy                       | ☐      |
| T0-2 | Config check               | notebook.enabled=true             | ☐      |
| T0-3 | Maestro connectivity       | HTTP 200                          | ☐      |
| T1-1 | Publish batch              | Kafka accepted                    | ☐      |
| T1-2 | Qdrant indexing            | 2 points created                  | ☐      |
| T1-3 | rag=false skip             | stg_post_003 absent               | ☐      |
| T2-1 | Publish insight            | Kafka accepted                    | ☐      |
| T2-2 | Insight in Qdrant          | 1 point in macro_insights         | ☐      |
| T2-3 | should_index=false skip    | trending_topic absent             | ☐      |
| T3-1 | Publish digest             | Kafka accepted                    | ☐      |
| T3-2 | Digest in Qdrant           | digest point exists               | ☐      |
| T3-3 | Notebook trigger logs      | BuildParts + SyncPart logged      | ☐      |
| T3-4 | Notebook DB records        | nb_campaigns + nb_sources created | ☐      |
| T4-1 | Webhook completed          | nb_sources.status = synced        | ☐      |
| T4-2 | Webhook failed             | nb_sources.status = failed        | ☐      |
| T5-1 | Structured chat            | HTTP 200, answer non-empty        | ☐      |
| T5-2 | Narrative without notebook | HTTP 200, fallback Qdrant         | ☐      |
| T6-1 | Async chat submit          | HTTP 202, chat_job_id             | ☐      |
| T6-2 | Poll job completed         | HTTP 200, status=completed        | ☐      |
| T6-3 | Webhook chat               | answer from webhook               | ☐      |
| T6-4 | Fallback on timeout        | HTTP 200, Qdrant answer           | ☐      |
| T7-1 | 404 on missing job         | HTTP 404                          | ☐      |
| T7-2 | should_index=false digest  | No notebook trigger               | ☐      |
| T7-3 | Idempotency                | point count unchanged             | ☐      |

---

## Định nghĩa Pass / Ship

**Go condition (ship to PROD):**

- T0 all pass
- T1-1, T1-2, T1-3 pass
- T2-1, T2-2 pass
- T3-1, T3-2 pass
- T5-1 pass (Qdrant chat baseline)
- T6-1, T6-2 pass (async flow end-to-end)

**Known deferred (không block ship):**

- T3-3, T3-4: Maestro live — chờ Maestro team confirm
- T4-x: Webhook simulation — accept nếu Maestro không live kịp
- T6-4: Fallback — accept nếu Maestro timeout behavior chưa test được

---

## Rollback Plan

Nếu bất kỳ **Go condition** nào fail:

1. **Revert config:** Đặt `notebook.enabled: false` → redeploy. Chat flow sẽ dùng 100% Qdrant, không ảnh hưởng user.
2. **Clean up Qdrant test data** (optional, không bắt buộc — STG data không nhạy cảm):

   ```bash
   curl -X DELETE http://qdrant.stg.internal:6333/collections/proj_stg_test_01
   ```

3. **Clean up PostgreSQL test data:**

   ```sql
   DELETE FROM nb_campaigns WHERE campaign_id IN ('camp_stg_q1', 'camp_stg_skip');
   DELETE FROM nb_sources WHERE campaign_id IN ('camp_stg_q1', 'camp_stg_skip');
   DELETE FROM nb_chat_jobs WHERE campaign_id IN ('camp_stg_q1', 'camp_stg_skip');
   ```

4. Báo cáo issue về channel `#knowledge-srv-eng` kèm logs.
