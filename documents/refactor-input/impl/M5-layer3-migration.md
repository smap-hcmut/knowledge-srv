# M5: Layer 3 Migration — InsightMessage Struct

**Prerequisite**: M1 + M2 + M3 + M4 phải done.

**⚠️ HIGHEST RISK MILESTONE** — Đây là consumer duy nhất đang hoạt động. Implement theo 2-phase để an toàn.

**Strategy**:
- **Phase A**: Thêm `IndexBatch()` method song song với `Index()` (giữ `Index()` hoàn toàn nguyên)
- **Phase B**: Worker detect format mới vs legacy → route đúng
- **Phase C** (sau khi analysis-srv migrate xong): Xóa legacy code

---

## Files cần thay đổi/tạo mới

### Phase A + B

#### 1. `internal/indexing/usecase/index_batch.go` ← FILE MỚI

```go
package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/indexing"
	kafkaDelivery "knowledge-srv/internal/indexing/delivery/kafka"
	repo "knowledge-srv/internal/indexing/repository"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// IndexBatch - Index a batch of InsightMessage documents from direct Kafka payload.
// Layer 3: analytics.batch.completed (new format).
// Không dùng MinIO, không dùng parseJSONL — documents[] đã có trong payload.
func (uc *implUseCase) IndexBatch(ctx context.Context, input indexing.IndexBatchInput) (indexing.IndexBatchOutput, error) {
	startTime := time.Now()

	if input.ProjectID == "" {
		return indexing.IndexBatchOutput{}, fmt.Errorf("IndexBatch: project_id is required")
	}
	if len(input.Documents) == 0 {
		return indexing.IndexBatchOutput{
			ProjectID:    input.ProjectID,
			TotalRecords: 0,
			Duration:     time.Since(startTime),
		}, nil
	}

	result := uc.processInsightBatch(ctx, input)
	result.Duration = time.Since(startTime)
	result.TotalRecords = len(input.Documents)

	// Invalidate search cache
	if result.Indexed > 0 {
		if err := uc.cacheRepo.InvalidateSearchCache(ctx, input.ProjectID); err != nil {
			uc.l.Warnf(ctx, "indexing.usecase.IndexBatch: Failed to invalidate cache: %v", err)
		}
	}

	uc.l.Infof(ctx, "indexing.usecase.IndexBatch: project=%s total=%d indexed=%d skipped=%d failed=%d duration=%s",
		input.ProjectID, result.TotalRecords, result.Indexed, result.Skipped, result.Failed, result.Duration)

	return result, nil
}

// processInsightBatch - xử lý documents[] song song với concurrency limit.
func (uc *implUseCase) processInsightBatch(ctx context.Context, input indexing.IndexBatchInput) indexing.IndexBatchOutput {
	var (
		indexed int
		failed  int
		skipped int
		mu      sync.Mutex
	)

	collectionName := fmt.Sprintf("proj_%s", input.ProjectID)

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(indexing.MaxConcurrency)

	for i := range input.Documents {
		doc := input.Documents[i]
		g.Go(func() error {
			result := uc.indexSingleInsight(gctx, input.ProjectID, input.CampaignID, collectionName, doc)
			mu.Lock()
			defer mu.Unlock()
			switch result {
			case "indexed":
				indexed++
			case "skipped":
				skipped++
			case "failed":
				failed++
			}
			return nil
		})
	}
	_ = g.Wait()

	return indexing.IndexBatchOutput{
		ProjectID: input.ProjectID,
		Indexed:   indexed,
		Skipped:   skipped,
		Failed:    failed,
	}
}

// indexSingleInsight - xử lý 1 InsightMessage: gate check → embed → upsert.
// Returns: "indexed" | "skipped" | "failed"
func (uc *implUseCase) indexSingleInsight(
	ctx context.Context,
	projectID string,
	campaignID string,
	collectionName string,
	doc indexing.InsightMessageInput,
) string {
	// Step 1: RAG gate — đây là gate duy nhất, không check spam/bot/quality
	if !doc.RAG {
		return "skipped"
	}

	// Step 2: Validate minimum required fields
	if doc.Identity.UapID == "" || doc.Content.CleanText == "" {
		uc.l.Warnf(ctx, "indexing.usecase.indexSingleInsight: skipping doc with empty uap_id or clean_text")
		return "skipped"
	}

	// Step 3: Embed clean_text
	genOutput, err := uc.embeddingUC.Generate(ctx, embedding.GenerateInput{
		Text: doc.Content.CleanText,
	})
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.indexSingleInsight: embedding failed for %s: %v", doc.Identity.UapID, err)
		return "failed"
	}

	// Step 4: Build Qdrant payload (nested fields → flat metadata)
	payload := uc.buildInsightPayload(projectID, campaignID, doc)

	// Step 5: Upsert — Point ID = uap_id
	err = uc.pointUC.Upsert(ctx, point.UpsertInput{
		CollectionName: collectionName,
		Points: []model.Point{
			{
				ID:      doc.Identity.UapID,
				Vector:  genOutput.Vector,
				Payload: payload,
			},
		},
	})
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.indexSingleInsight: qdrant upsert failed for %s: %v", doc.Identity.UapID, err)
		return "failed"
	}

	return "indexed"
}

// buildInsightPayload - flat map từ InsightMessageInput cho Qdrant payload.
// Tất cả fields từ contract.md Section 2 đều được lưu vào metadata.
func (uc *implUseCase) buildInsightPayload(
	projectID string,
	campaignID string,
	doc indexing.InsightMessageInput,
) map[string]interface{} {
	// Serialize aspects và entities thành arrays
	aspects := make([]map[string]interface{}, len(doc.NLP.Aspects))
	for i, a := range doc.NLP.Aspects {
		aspects[i] = map[string]interface{}{
			"aspect":   a.Aspect,
			"polarity": a.Polarity,
		}
	}

	entities := make([]map[string]interface{}, len(doc.NLP.Entities))
	for i, e := range doc.NLP.Entities {
		entities[i] = map[string]interface{}{
			"type":  e.Type,
			"value": e.Value,
		}
	}

	return map[string]interface{}{
		// Scope
		"project_id":  projectID,
		"campaign_id": campaignID,

		// Identity
		"uap_id":         doc.Identity.UapID,
		"uap_type":       doc.Identity.UapType,
		"uap_media_type": doc.Identity.UapMediaType,
		"platform":       doc.Identity.Platform,
		"published_at":   doc.Identity.PublishedAt,

		// Content
		"content_summary": doc.Content.Summary,

		// NLP
		"sentiment_label": doc.NLP.Sentiment.Label,
		"sentiment_score": doc.NLP.Sentiment.Score,
		"aspects":         aspects,
		"entities":        entities,

		// Business
		"impact_score": doc.Business.Impact.ImpactScore,
		"priority":     doc.Business.Impact.Priority,
		"likes":        doc.Business.Impact.Engagement.Likes,
		"comments":     doc.Business.Impact.Engagement.Comments,
		"shares":       doc.Business.Impact.Engagement.Shares,
		"views":        doc.Business.Impact.Engagement.Views,
	}
}

// generateInsightContentHash - hash clean_text để dedup (dùng nếu cần).
func generateInsightContentHash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}
```

---

#### 2. `internal/indexing/delivery/kafka/consumer/workers.go`

**REPLACE** `handleBatchCompletedMessage` — thêm format detection, giữ legacy path:

```go
// handleBatchCompletedMessage receives Layer 3 message.
// Detects message format: new (documents[]) vs legacy (file_url) và route đúng.
func (c *consumer) handleBatchCompletedMessage(msg *sarama.ConsumerMessage) error {
	ctx := context.Background()

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: Processing message from partition %d, offset %d",
		msg.Partition, msg.Offset)

	// 1. Try new format first (BatchCompletedMessage với documents[])
	var newMsg kafka.BatchCompletedMessage
	if err := json.Unmarshal(msg.Value, &newMsg); err == nil && len(newMsg.Documents) > 0 {
		c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: Detected new format (documents[]), project=%s docs=%d",
			newMsg.ProjectID, len(newMsg.Documents))
		return c.handleNewBatchCompleted(ctx, newMsg)
	}

	// 2. Fallback to legacy format (LegacyBatchCompletedMessage với file_url)
	var legacyMsg kafka.LegacyBatchCompletedMessage
	if err := json.Unmarshal(msg.Value, &legacyMsg); err == nil && legacyMsg.FileURL != "" {
		c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: Detected legacy format (file_url), batch=%s",
			legacyMsg.BatchID)
		return c.handleLegacyBatchCompleted(ctx, legacyMsg)
	}

	// 3. Invalid — skip
	c.l.Warnf(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: Cannot parse message (neither new nor legacy format), skipping")
	return nil
}

// handleNewBatchCompleted - xử lý format mới với documents[] array.
func (c *consumer) handleNewBatchCompleted(ctx context.Context, message kafka.BatchCompletedMessage) error {
	if message.ProjectID == "" {
		c.l.Warnf(ctx, "indexing.delivery.kafka.consumer.handleNewBatchCompleted: Missing project_id (skipping)")
		return nil
	}

	input := toIndexBatchInput(message)

	sc := auth.Scope{UserID: "system", Role: "system"}
	ctx = auth.SetScopeToContext(ctx, sc)

	output, err := c.uc.IndexBatch(ctx, input)
	if err != nil {
		c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.handleNewBatchCompleted: IndexBatch failed: %v", err)
		return fmt.Errorf("usecase error: %w", err)
	}

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleNewBatchCompleted: project=%s total=%d indexed=%d skipped=%d failed=%d",
		message.ProjectID, output.TotalRecords, output.Indexed, output.Skipped, output.Failed)
	return nil
}

// handleLegacyBatchCompleted - xử lý format cũ với file_url (deprecated).
// Giữ nguyên logic cũ để backward compat. Sẽ xóa ở Phase C.
func (c *consumer) handleLegacyBatchCompleted(ctx context.Context, message kafka.LegacyBatchCompletedMessage) error {
	if message.BatchID == "" || message.FileURL == "" {
		c.l.Warnf(ctx, "indexing.delivery.kafka.consumer.handleLegacyBatchCompleted: Missing required fields (skipping)")
		return nil
	}

	input := toIndexInput(message)

	sc := auth.Scope{UserID: "system", Role: "system"}
	ctx = auth.SetScopeToContext(ctx, sc)

	output, err := c.uc.Index(ctx, input)
	if err != nil {
		c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.handleLegacyBatchCompleted: Index failed: %v", err)
		return fmt.Errorf("usecase error: %w", err)
	}

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleLegacyBatchCompleted: batch=%s indexed=%d failed=%d skipped=%d",
		message.BatchID, output.Indexed, output.Failed, output.Skipped)
	return nil
}
```

**Lưu ý**: Xóa function `handleBatchCompletedMessage` cũ và thay bằng 3 functions mới ở trên. Giữ nguyên `handleInsightsPublishedMessage` và `handleReportDigestMessage`.

---

#### 3. `internal/indexing/delivery/kafka/consumer/presenters.go`

**THAY ĐỔI** `toIndexInput` để nhận `LegacyBatchCompletedMessage` (tên type đã đổi ở M2). **APPEND** `toIndexBatchInput`:

```go
// toIndexInput maps LEGACY Kafka message → usecase input.
// Deprecated — chỉ dùng cho backward compat với format cũ (file_url).
func toIndexInput(m kafkaDelivery.LegacyBatchCompletedMessage) indexing.IndexInput {
	return indexing.IndexInput{
		BatchID:     m.BatchID,
		ProjectID:   m.ProjectID,
		CampaignID:  m.CampaignID,
		FileURL:     m.FileURL,
		RecordCount: m.RecordCount,
	}
}

// toIndexBatchInput maps new Kafka BatchCompletedMessage → usecase input.
func toIndexBatchInput(m kafkaDelivery.BatchCompletedMessage) indexing.IndexBatchInput {
	docs := make([]indexing.InsightMessageInput, len(m.Documents))
	for i, d := range m.Documents {
		aspects := make([]indexing.InsightAspectInput, len(d.NLP.Aspects))
		for j, a := range d.NLP.Aspects {
			aspects[j] = indexing.InsightAspectInput{
				Aspect:   a.Aspect,
				Polarity: a.Polarity,
			}
		}

		entities := make([]indexing.InsightEntityInput, len(d.NLP.Entities))
		for j, e := range d.NLP.Entities {
			entities[j] = indexing.InsightEntityInput{
				Type:  e.Type,
				Value: e.Value,
			}
		}

		docs[i] = indexing.InsightMessageInput{
			Identity: indexing.InsightIdentityInput{
				UapID:        d.Identity.UapID,
				UapType:      d.Identity.UapType,
				UapMediaType: d.Identity.UapMediaType,
				Platform:     d.Identity.Platform,
				PublishedAt:  d.Identity.PublishedAt,
			},
			Content: indexing.InsightContentInput{
				CleanText: d.Content.CleanText,
				Summary:   d.Content.Summary,
			},
			NLP: indexing.InsightNLPInput{
				Sentiment: indexing.InsightSentimentInput{
					Label: d.NLP.Sentiment.Label,
					Score: d.NLP.Sentiment.Score,
				},
				Aspects:  aspects,
				Entities: entities,
			},
			Business: indexing.InsightBusinessInput{
				Impact: indexing.InsightImpactInput{
					Engagement: indexing.InsightEngagementInput{
						Likes:    d.Business.Impact.Engagement.Likes,
						Comments: d.Business.Impact.Engagement.Comments,
						Shares:   d.Business.Impact.Engagement.Shares,
						Views:    d.Business.Impact.Engagement.Views,
					},
					ImpactScore: d.Business.Impact.ImpactScore,
					Priority:    d.Business.Impact.Priority,
				},
			},
			RAG: d.RAG,
		}
	}

	return indexing.IndexBatchInput{
		ProjectID:  m.ProjectID,
		CampaignID: m.CampaignID,
		Documents:  docs,
	}
}
```

---

#### 4. `config/kafka/consumer.go` — MaxMessageBytes

Đọc file này và đảm bảo sarama config cho consumer có `MaxMessageBytes` đủ lớn cho ~4MB payload:

```go
// Thêm vào sarama config khi tạo consumer cho topic analytics.batch.completed:
saramaConfig.Consumer.Fetch.Default = 1024 * 1024 * 10   // 10MB default fetch
saramaConfig.Consumer.MaxWaitTime = 250 * time.Millisecond
```

Xem file hiện tại trước khi sửa — tìm chỗ configure sarama và thêm vào đó.

---

## Phase C: Cleanup (chỉ làm sau khi analysis-srv xác nhận đã migrate hoàn toàn)

Khi analysis-srv không còn send legacy format, xóa:

1. `LegacyBatchCompletedMessage` struct từ `type.go`
2. `handleLegacyBatchCompleted()` từ `workers.go`
3. `toIndexInput()` từ `presenters.go`
4. `Index()` method từ `interface.go` và `usecase/index.go`
5. `AnalyticsPost` struct từ `types.go` (sau khi không còn referenced)
6. MinIO import từ `usecase/new.go` constructor

---

## Verification Checklist

```bash
# 1. Build pass
go build ./...

# 2. Test new format:
# Construct BatchCompletedMessage với documents[] (theo contract.md Section 1)
# Produce vào analytics.batch.completed
# Expected: N points trong proj_{project_id} với rag=true
# Expected: documents với rag=false bị skip

# 3. Test legacy format backward compat:
# Produce message với format cũ (batch_id, file_url)
# Expected: vẫn process bình thường qua legacy path

# 4. Kiểm tra Qdrant point có đúng fields:
# - project_id, campaign_id, uap_id, uap_type, platform, published_at
# - sentiment_label, sentiment_score
# - aspects[], entities[]
# - impact_score, priority
# - content_summary

# 5. Kiểm tra collection name: proj_{project_id} (không phải smap_analytics)
```

## Notes cho reviewer

- **Format detection logic** trong `handleBatchCompletedMessage`: detect bằng `len(newMsg.Documents) > 0` thay vì `newMsg.FileURL != ""` — cần đảm bảo JSON unmarshal không error với wrong format
- `IndexBatch` không dùng PostgreSQL tracking để tránh phức tạp — nếu cần tracking thì thêm vào Phase C
- `buildInsightPayload` flatten nested struct → flat map: reviewer phải verify đủ fields so với contract.md Section 2
- `MaxConcurrency` constant đã có trong `indexing/types.go` — giữ nguyên
- `generateInsightContentHash` helper ở cuối file — có thể dùng cho dedup sau này nếu cần
