# M3: Layer 2 Consumer — Insight Cards

**Prerequisite**: M1 + M2 phải done.

**Goal**: End-to-end flow: Kafka `analytics.insights.published` → Qdrant `macro_insights`.

**Implement trước M5** vì đây là pattern đơn giản nhất — agent học convention từ đây trước khi đụng Layer 3.

---

## Files cần thay đổi/tạo mới

### 1. `internal/indexing/usecase/index_insight.go` ← FILE MỚI

```go
package usecase

import (
	"context"
	"crypto/sha256"
	"fmt"
	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/indexing"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
	kafkaDelivery "knowledge-srv/internal/indexing/delivery/kafka"
	"time"
)

// IndexInsight - Index 1 insight card vào Qdrant collection macro_insights.
// Layer 2: analytics.insights.published
func (uc *implUseCase) IndexInsight(ctx context.Context, input indexing.IndexInsightInput) (indexing.IndexInsightOutput, error) {
	startTime := time.Now()

	// Step 1: Build embed text (title + summary)
	embedText := input.Title + ". " + input.Summary

	// Step 2: Embed via embedding domain (cached by Redis)
	genOutput, err := uc.embeddingUC.Generate(ctx, embedding.GenerateInput{
		Text: embedText,
	})
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.IndexInsight: embedding failed: %v", err)
		return indexing.IndexInsightOutput{}, fmt.Errorf("%w: %v", indexing.ErrEmbeddingFailed, err)
	}

	// Step 3: Generate Point ID
	// Format: insight:{run_id}:{insight_type}:{sha256(title)[:12]}
	hash := sha256.Sum256([]byte(input.Title))
	pointID := fmt.Sprintf("insight:%s:%s:%x", input.RunID, input.InsightType, hash[:6])

	// Step 4: Build Qdrant payload
	payload := map[string]interface{}{
		"project_id":             input.ProjectID,
		"campaign_id":            input.CampaignID,
		"run_id":                 input.RunID,
		"rag_document_type":      "insight_card", // hardcoded — knowledge-srv tự gán
		"insight_type":           input.InsightType,
		"title":                  input.Title,
		"summary":                input.Summary,
		"confidence":             input.Confidence,
		"analysis_window_start":  input.AnalysisWindowStart,
		"analysis_window_end":    input.AnalysisWindowEnd,
		"supporting_metrics":     input.SupportingMetrics,
		"evidence_references":    input.EvidenceReferences,
	}

	// Step 5: Upsert vào macro_insights
	err = uc.pointUC.Upsert(ctx, point.UpsertInput{
		CollectionName: kafkaDelivery.CollectionMacroInsights,
		Points: []model.Point{
			{
				ID:      pointID,
				Vector:  genOutput.Vector,
				Payload: payload,
			},
		},
	})
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.IndexInsight: qdrant upsert failed: %v", err)
		return indexing.IndexInsightOutput{}, fmt.Errorf("%w: %v", indexing.ErrQdrantUpsertFailed, err)
	}

	uc.l.Infof(ctx, "indexing.usecase.IndexInsight: indexed insight %s (point: %s, duration: %s)",
		input.InsightType, pointID, time.Since(startTime))

	return indexing.IndexInsightOutput{
		PointID:  pointID,
		Duration: time.Since(startTime),
	}, nil
}
```

---

### 2. `internal/indexing/delivery/kafka/consumer/new.go`

**REPLACE toàn bộ file** — mở rộng `Consumer` interface + thêm group fields vào struct:

```go
package consumer

import (
	"context"
	"fmt"
	"knowledge-srv/config"
	"knowledge-srv/internal/indexing"

	"github.com/smap-hcmut/shared-libs/go/kafka"
	"github.com/smap-hcmut/shared-libs/go/log"
)

// Consumer is the delivery interface for Kafka.
type Consumer interface {
	ConsumeBatchCompleted(ctx context.Context) error
	ConsumeInsightsPublished(ctx context.Context) error // NEW Layer 2
	ConsumeReportDigest(ctx context.Context) error      // NEW Layer 1
	Close() error
}

type Config struct {
	Logger      log.Logger
	KafkaConfig config.KafkaConfig
	UseCase     indexing.UseCase
}

// consumer implements Consumer.
type consumer struct {
	l           log.Logger
	kafkaConfig config.KafkaConfig
	uc          indexing.UseCase

	batchCompletedGroup    kafka.IConsumer
	insightsPublishedGroup kafka.IConsumer // NEW
	reportDigestGroup      kafka.IConsumer // NEW
}

func New(cfg Config) (Consumer, error) {
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if cfg.UseCase == nil {
		return nil, fmt.Errorf("usecase is required")
	}
	if len(cfg.KafkaConfig.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}

	return &consumer{
		l:           cfg.Logger,
		kafkaConfig: cfg.KafkaConfig,
		uc:          cfg.UseCase,
	}, nil
}

func (c *consumer) Close() error {
	ctx := context.Background()
	var firstErr error

	if c.batchCompletedGroup != nil {
		if err := c.batchCompletedGroup.Close(); err != nil {
			c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.Close: failed to close batchCompleted group: %v", err)
			if firstErr == nil {
				firstErr = ErrConsumerGroupNotFound
			}
		}
	}
	if c.insightsPublishedGroup != nil {
		if err := c.insightsPublishedGroup.Close(); err != nil {
			c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.Close: failed to close insightsPublished group: %v", err)
			if firstErr == nil {
				firstErr = ErrConsumerGroupNotFound
			}
		}
	}
	if c.reportDigestGroup != nil {
		if err := c.reportDigestGroup.Close(); err != nil {
			c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.Close: failed to close reportDigest group: %v", err)
			if firstErr == nil {
				firstErr = ErrConsumerGroupNotFound
			}
		}
	}

	return firstErr
}

func (c *consumer) createConsumerGroup(groupID string) (kafka.IConsumer, error) {
	consumerConfig := kafka.ConsumerConfig{
		Brokers: c.kafkaConfig.Brokers,
		GroupID: groupID,
	}
	group, err := kafka.NewConsumer(consumerConfig)
	if err != nil {
		c.l.Errorf(context.Background(), "indexing.delivery.kafka.consumer.createConsumerGroup: failed to create consumer group %s: %v", groupID, err)
		return nil, ErrCreateConsumerGroupFailed
	}
	return group, nil
}
```

---

### 3. `internal/indexing/delivery/kafka/consumer/consumer.go`

**APPEND** method `ConsumeInsightsPublished` vào file (giữ nguyên `ConsumeBatchCompleted`):

```go
// ConsumeInsightsPublished starts consuming analytics.insights.published messages (Layer 2)
func (c *consumer) ConsumeInsightsPublished(ctx context.Context) error {
	group, err := c.createConsumerGroup(kafka.GroupIDInsightsPublished)
	if err != nil {
		return err
	}
	c.insightsPublishedGroup = group

	handler := &insightsPublishedHandler{consumer: c}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := group.ConsumeWithContext(ctx, []string{kafka.TopicInsightsPublished}, handler); err != nil {
					c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.ConsumeInsightsPublished: Consumer error: %v", err)
				}
			}
		}
	}()

	go func() {
		for err := range group.Errors() {
			c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.ConsumeInsightsPublished: Consumer group error: %v", err)
		}
	}()

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.ConsumeInsightsPublished: Started consuming topic: %s (group: %s)",
		kafka.TopicInsightsPublished, kafka.GroupIDInsightsPublished)

	return nil
}
```

**Lưu ý**: `consumer.go` hiện chỉ có `ConsumeBatchCompleted`. Append thêm method này vào cuối file. Import `kafka "knowledge-srv/internal/indexing/delivery/kafka"` đã có sẵn.

---

### 4. `internal/indexing/delivery/kafka/consumer/handler.go`

**APPEND** `insightsPublishedHandler` vào cuối file (giữ `batchCompletedHandler`):

```go
// insightsPublishedHandler implements sarama.ConsumerGroupHandler for Layer 2
type insightsPublishedHandler struct {
	consumer *consumer
}

func (h *insightsPublishedHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *insightsPublishedHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }
func (h *insightsPublishedHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.consumer.handleInsightsPublishedMessage(msg); err != nil {
			h.consumer.l.Errorf(context.Background(),
				"indexing.delivery.kafka.consumer.ConsumeInsightsPublished: Failed to process message: %v", err)
			continue
		}
		session.MarkMessage(msg, "")
	}
	return nil
}
```

---

### 5. `internal/indexing/delivery/kafka/consumer/workers.go`

**APPEND** `handleInsightsPublishedMessage` vào cuối file (giữ `handleBatchCompletedMessage`):

```go
// handleInsightsPublishedMessage receives Layer 2 message, validates, delegates to usecase.
func (c *consumer) handleInsightsPublishedMessage(msg *sarama.ConsumerMessage) error {
	ctx := context.Background()

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleInsightsPublishedMessage: Processing message from partition %d, offset %d",
		msg.Partition, msg.Offset)

	// 1. Unmarshal
	var message kafka.InsightsPublishedMessage
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		c.l.Warnf(ctx, "indexing.delivery.kafka.consumer.handleInsightsPublishedMessage: Invalid message format (skipping): %v", err)
		return nil // skip — commit offset
	}

	// 2. Gate check
	if !message.ShouldIndex {
		c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleInsightsPublishedMessage: should_index=false, skipping insight %s/%s",
			message.RunID, message.InsightType)
		return nil // skip — commit offset
	}

	// 3. Validate required fields
	if message.ProjectID == "" || message.RunID == "" || message.InsightType == "" || message.Title == "" {
		c.l.Warnf(ctx, "indexing.delivery.kafka.consumer.handleInsightsPublishedMessage: Missing required fields (skipping)")
		return nil // skip — commit offset
	}

	// 4. Map to usecase input
	input := toIndexInsightInput(message)

	// 5. Set system scope
	sc := auth.Scope{UserID: "system", Role: "system"}
	ctx = auth.SetScopeToContext(ctx, sc)

	// 6. Call usecase
	output, err := c.uc.IndexInsight(ctx, input)
	if err != nil {
		c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.handleInsightsPublishedMessage: IndexInsight failed: %v", err)
		return fmt.Errorf("usecase error: %w", err)
	}

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleInsightsPublishedMessage: Indexed insight %s (point: %s, duration: %s)",
		message.InsightType, output.PointID, output.Duration)
	return nil
}
```

**Import** cần thêm vào `workers.go`: file hiện đã có `"encoding/json"`, `"context"`, `auth`, `sarama`. Cần đảm bảo `"fmt"` cũng có.

---

### 6. `internal/indexing/delivery/kafka/consumer/presenters.go`

**APPEND** `toIndexInsightInput` vào cuối file (giữ `toIndexInput`):

```go
// toIndexInsightInput maps Layer 2 Kafka message → usecase input.
func toIndexInsightInput(m kafkaDelivery.InsightsPublishedMessage) indexing.IndexInsightInput {
	return indexing.IndexInsightInput{
		ProjectID:           m.ProjectID,
		CampaignID:          m.CampaignID,
		RunID:               m.RunID,
		InsightType:         m.InsightType,
		Title:               m.Title,
		Summary:             m.Summary,
		Confidence:          m.Confidence,
		AnalysisWindowStart: m.AnalysisWindowStart,
		AnalysisWindowEnd:   m.AnalysisWindowEnd,
		SupportingMetrics:   m.SupportingMetrics,
		EvidenceReferences:  m.EvidenceReferences,
	}
}
```

---

### 7. `internal/consumer/handler.go`

Tìm `startConsumers()` và `stopConsumers()`. **Thêm** Layer 2 consumer vào cả 2 functions:

**`startConsumers()`** — thêm sau dòng `ConsumeBatchCompleted`:

```go
func (srv *ConsumerServer) startConsumers(ctx context.Context, consumers *domainConsumers) error {
	if err := consumers.indexingConsumer.ConsumeBatchCompleted(ctx); err != nil {
		return fmt.Errorf("failed to start batch consumer: %w", err)
	}
	// NEW
	if err := consumers.indexingConsumer.ConsumeInsightsPublished(ctx); err != nil {
		return fmt.Errorf("failed to start insights consumer: %w", err)
	}
	// ConsumeReportDigest sẽ thêm ở M4

	srv.l.Infof(ctx, "All consumers started successfully")
	return nil
}
```

**`stopConsumers()`** — không cần thay đổi vì `Close()` đã xử lý tất cả groups.

---

## Verification Checklist

```bash
# 1. Build pass
go build ./...

# 2. Chạy consumer, observe logs:
# → "Started consuming topic: analytics.insights.published (group: knowledge-indexing-insights)"

# 3. Produce test message từ sample file:
# documents/refactor-input/outputSRM/insights/insights.jsonl
# Mỗi line là 1 JSON → produce vào topic analytics.insights.published
# Expected: 7 points xuất hiện trong Qdrant collection macro_insights

# 4. Kiểm tra Qdrant point có đúng fields:
# - rag_document_type = "insight_card"
# - project_id, campaign_id, run_id đúng
# - Point ID format: insight:{run_id}:{insight_type}:{12-char hex}

# 5. Produce message với should_index=false → không có point mới

# 6. Produce message thiếu field title="" → không có point mới, log WARN
```

## Notes cho reviewer

- `IndexInsight` không dùng PostgreSQL tracking (indexed_documents) vì Layer 2 không cần dedup hay retry tracking qua DB — chỉ Qdrant upsert
- Error từ usecase → `return fmt.Errorf(...)` (không nil) → offset KHÔNG commit → Kafka retry
- Embedding cache qua Redis tự động (embeddingUC.Generate đã có Redis cache)
- Upsert strategy: `insight:{run_id}:{insight_type}:{hash(title)[:12]}` — same run_id + insight_type sẽ overwrite point cũ
