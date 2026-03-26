# M4: Layer 1 Consumer — Report Digest

**Prerequisite**: M1 + M2 + M3 phải done.

**Goal**: End-to-end flow: Kafka `analytics.report.digest` → generate prose → embed → Qdrant `macro_insights`.

**Có thể parallel với M3** nhưng nên implement sau M3 vì pattern giống hệt — chỉ thêm bước `buildDigestProse`.

---

## Files cần thay đổi/tạo mới

### 1. `internal/indexing/usecase/index_digest.go` ← FILE MỚI

```go
package usecase

import (
	"context"
	"fmt"
	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/indexing"
	kafkaDelivery "knowledge-srv/internal/indexing/delivery/kafka"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
	"strings"
	"time"
)

// IndexDigest - Index 1 report digest vào Qdrant collection macro_insights.
// Layer 1: analytics.report.digest
func (uc *implUseCase) IndexDigest(ctx context.Context, input indexing.IndexDigestInput) (indexing.IndexDigestOutput, error) {
	startTime := time.Now()

	// Step 1: Build prose text từ structured data
	prose := uc.buildDigestProse(input)
	if prose == "" {
		return indexing.IndexDigestOutput{}, indexing.ErrDigestBuildFailed
	}

	// Step 2: Embed prose
	genOutput, err := uc.embeddingUC.Generate(ctx, embedding.GenerateInput{
		Text: prose,
	})
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.IndexDigest: embedding failed: %v", err)
		return indexing.IndexDigestOutput{}, fmt.Errorf("%w: %v", indexing.ErrEmbeddingFailed, err)
	}

	// Step 3: Generate Point ID
	// Format: digest:{run_id} — unique per run, upsert thay thế digest cũ cùng run
	pointID := fmt.Sprintf("digest:%s", input.RunID)

	// Step 4: Build payload — lưu full structured arrays để agent có thể drill-down
	payload := map[string]interface{}{
		"project_id":            input.ProjectID,
		"campaign_id":           input.CampaignID,
		"run_id":                input.RunID,
		"rag_document_type":     "report_digest", // hardcoded — knowledge-srv tự gán
		"analysis_window_start": input.AnalysisWindowStart,
		"analysis_window_end":   input.AnalysisWindowEnd,
		"domain_overlay":        input.DomainOverlay,
		"platform":              input.Platform,
		"total_mentions":        input.TotalMentions,
		"top_entities":          input.TopEntities,
		"top_topics":            input.TopTopics,
		"top_issues":            input.TopIssues,
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
		uc.l.Errorf(ctx, "indexing.usecase.IndexDigest: qdrant upsert failed: %v", err)
		return indexing.IndexDigestOutput{}, fmt.Errorf("%w: %v", indexing.ErrQdrantUpsertFailed, err)
	}

	uc.l.Infof(ctx, "indexing.usecase.IndexDigest: indexed digest for run %s (point: %s, duration: %s)",
		input.RunID, pointID, time.Since(startTime))

	return indexing.IndexDigestOutput{
		PointID:  pointID,
		Duration: time.Since(startTime),
	}, nil
}

// buildDigestProse builds human-readable prose text từ structured digest data.
// Text này sẽ được embed → vector cho RAG retrieval.
// Format đủ semantic để agent tìm được khi hỏi về campaign overview.
func (uc *implUseCase) buildDigestProse(input indexing.IndexDigestInput) string {
	var b strings.Builder

	// Header
	fmt.Fprintf(&b, "Campaign Report: %s\n", input.DomainOverlay)
	fmt.Fprintf(&b, "Platform: %s | Total Mentions: %d\n", input.Platform, input.TotalMentions)
	fmt.Fprintf(&b, "Analysis Window: %s to %s\n\n", input.AnalysisWindowStart, input.AnalysisWindowEnd)

	// Top Entities (tối đa 5)
	if len(input.TopEntities) > 0 {
		b.WriteString("Top Brands:\n")
		limit := len(input.TopEntities)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			e := input.TopEntities[i]
			fmt.Fprintf(&b, "- %s: %d mentions (%.1f%% share)\n",
				e.EntityName, e.MentionCount, e.MentionShare*100)
		}
		b.WriteString("\n")
	}

	// Top Topics (tối đa 5)
	if len(input.TopTopics) > 0 {
		b.WriteString("Key Discussion Topics:\n")
		limit := len(input.TopTopics)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			t := input.TopTopics[i]
			fmt.Fprintf(&b, "- %s: %d mentions", t.TopicLabel, t.MentionCount)
			if t.QualityScore != nil {
				fmt.Fprintf(&b, " (quality: %.2f)", *t.QualityScore)
			}
			b.WriteString("\n")
			if len(t.RepresentativeTexts) > 0 {
				fmt.Fprintf(&b, "  Example: \"%s\"\n", t.RepresentativeTexts[0])
			}
		}
		b.WriteString("\n")
	}

	// Top Issues (tối đa 5)
	if len(input.TopIssues) > 0 {
		b.WriteString("Critical Issues:\n")
		limit := len(input.TopIssues)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			issue := input.TopIssues[i]
			fmt.Fprintf(&b, "- %s: %d mentions (pressure: %.2f)\n",
				issue.IssueCategory, issue.MentionCount, issue.IssuePressureProxy)
		}
	}

	return b.String()
}
```

---

### 2. `internal/indexing/delivery/kafka/consumer/consumer.go`

**APPEND** `ConsumeReportDigest` vào cuối file (sau `ConsumeInsightsPublished` từ M3):

```go
// ConsumeReportDigest starts consuming analytics.report.digest messages (Layer 1)
func (c *consumer) ConsumeReportDigest(ctx context.Context) error {
	group, err := c.createConsumerGroup(kafka.GroupIDReportDigest)
	if err != nil {
		return err
	}
	c.reportDigestGroup = group

	handler := &reportDigestHandler{consumer: c}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := group.ConsumeWithContext(ctx, []string{kafka.TopicReportDigest}, handler); err != nil {
					c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.ConsumeReportDigest: Consumer error: %v", err)
				}
			}
		}
	}()

	go func() {
		for err := range group.Errors() {
			c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.ConsumeReportDigest: Consumer group error: %v", err)
		}
	}()

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.ConsumeReportDigest: Started consuming topic: %s (group: %s)",
		kafka.TopicReportDigest, kafka.GroupIDReportDigest)

	return nil
}
```

---

### 3. `internal/indexing/delivery/kafka/consumer/handler.go`

**APPEND** `reportDigestHandler` vào cuối file:

```go
// reportDigestHandler implements sarama.ConsumerGroupHandler for Layer 1
type reportDigestHandler struct {
	consumer *consumer
}

func (h *reportDigestHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *reportDigestHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }
func (h *reportDigestHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.consumer.handleReportDigestMessage(msg); err != nil {
			h.consumer.l.Errorf(context.Background(),
				"indexing.delivery.kafka.consumer.ConsumeReportDigest: Failed to process message: %v", err)
			continue
		}
		session.MarkMessage(msg, "")
	}
	return nil
}
```

---

### 4. `internal/indexing/delivery/kafka/consumer/workers.go`

**APPEND** `handleReportDigestMessage` vào cuối file:

```go
// handleReportDigestMessage receives Layer 1 message, validates, delegates to usecase.
func (c *consumer) handleReportDigestMessage(msg *sarama.ConsumerMessage) error {
	ctx := context.Background()

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleReportDigestMessage: Processing message from partition %d, offset %d",
		msg.Partition, msg.Offset)

	// 1. Unmarshal
	var message kafka.ReportDigestMessage
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		c.l.Warnf(ctx, "indexing.delivery.kafka.consumer.handleReportDigestMessage: Invalid message format (skipping): %v", err)
		return nil // skip — commit offset
	}

	// 2. Gate check
	if !message.ShouldIndex {
		c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleReportDigestMessage: should_index=false, skipping run %s", message.RunID)
		return nil // skip — commit offset
	}

	// 3. Validate required fields
	if message.ProjectID == "" || message.RunID == "" || message.DomainOverlay == "" {
		c.l.Warnf(ctx, "indexing.delivery.kafka.consumer.handleReportDigestMessage: Missing required fields (skipping)")
		return nil // skip — commit offset
	}

	// 4. Map to usecase input
	input := toIndexDigestInput(message)

	// 5. Set system scope
	sc := auth.Scope{UserID: "system", Role: "system"}
	ctx = auth.SetScopeToContext(ctx, sc)

	// 6. Call usecase
	output, err := c.uc.IndexDigest(ctx, input)
	if err != nil {
		c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.handleReportDigestMessage: IndexDigest failed: %v", err)
		return fmt.Errorf("usecase error: %w", err)
	}

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleReportDigestMessage: Indexed digest run %s (point: %s, duration: %s)",
		message.RunID, output.PointID, output.Duration)
	return nil
}
```

---

### 5. `internal/indexing/delivery/kafka/consumer/presenters.go`

**APPEND** `toIndexDigestInput` vào cuối file:

```go
// toIndexDigestInput maps Layer 1 Kafka message → usecase input.
func toIndexDigestInput(m kafkaDelivery.ReportDigestMessage) indexing.IndexDigestInput {
	entities := make([]indexing.TopEntityInput, len(m.TopEntities))
	for i, e := range m.TopEntities {
		entities[i] = indexing.TopEntityInput{
			CanonicalEntityID: e.CanonicalEntityID,
			EntityName:        e.EntityName,
			EntityType:        e.EntityType,
			MentionCount:      e.MentionCount,
			MentionShare:      e.MentionShare,
		}
	}

	topics := make([]indexing.TopTopicInput, len(m.TopTopics))
	for i, t := range m.TopTopics {
		topics[i] = indexing.TopTopicInput{
			TopicKey:            t.TopicKey,
			TopicLabel:          t.TopicLabel,
			MentionCount:        t.MentionCount,
			MentionShare:        t.MentionShare,
			BuzzScoreProxy:      t.BuzzScoreProxy,
			QualityScore:        t.QualityScore,
			RepresentativeTexts: t.RepresentativeTexts,
		}
	}

	issues := make([]indexing.TopIssueInput, len(m.TopIssues))
	for i, iss := range m.TopIssues {
		var severityMix *indexing.SeverityMixInput
		if iss.SeverityMix != nil {
			severityMix = &indexing.SeverityMixInput{
				Low:    iss.SeverityMix.Low,
				Medium: iss.SeverityMix.Medium,
				High:   iss.SeverityMix.High,
			}
		}
		issues[i] = indexing.TopIssueInput{
			IssueCategory:      iss.IssueCategory,
			MentionCount:       iss.MentionCount,
			IssuePressureProxy: iss.IssuePressureProxy,
			SeverityMix:        severityMix,
		}
	}

	return indexing.IndexDigestInput{
		ProjectID:           m.ProjectID,
		CampaignID:          m.CampaignID,
		RunID:               m.RunID,
		AnalysisWindowStart: m.AnalysisWindowStart,
		AnalysisWindowEnd:   m.AnalysisWindowEnd,
		DomainOverlay:       m.DomainOverlay,
		Platform:            m.Platform,
		TotalMentions:       m.TotalMentions,
		TopEntities:         entities,
		TopTopics:           topics,
		TopIssues:           issues,
	}
}
```

---

### 6. `internal/consumer/handler.go`

Update `startConsumers()` thêm `ConsumeReportDigest`:

```go
func (srv *ConsumerServer) startConsumers(ctx context.Context, consumers *domainConsumers) error {
	if err := consumers.indexingConsumer.ConsumeBatchCompleted(ctx); err != nil {
		return fmt.Errorf("failed to start batch consumer: %w", err)
	}
	if err := consumers.indexingConsumer.ConsumeInsightsPublished(ctx); err != nil {
		return fmt.Errorf("failed to start insights consumer: %w", err)
	}
	// NEW
	if err := consumers.indexingConsumer.ConsumeReportDigest(ctx); err != nil {
		return fmt.Errorf("failed to start digest consumer: %w", err)
	}

	srv.l.Infof(ctx, "All consumers started successfully")
	return nil
}
```

---

## Verification Checklist

```bash
# 1. Build pass
go build ./...

# 2. Chạy consumer, observe logs:
# → "Started consuming topic: analytics.report.digest (group: knowledge-indexing-digest)"

# 3. Produce test digest message (construct từ contract.md Section 4 example)
# Expected: 1 point xuất hiện trong macro_insights

# 4. Kiểm tra point fields:
# - rag_document_type = "report_digest"
# - Point ID = "digest:{run_id}"
# - top_entities, top_topics, top_issues có đầy đủ

# 5. Kiểm tra prose text có đúng format:
# - Chứa domain_overlay, platform, total_mentions
# - Chứa entity names + mention counts
# - Chứa topic labels
# - Chứa issue categories

# 6. Produce message với should_index=false → không có point mới
```

## Notes cho reviewer

- `buildDigestProse` cần test manually: prose text phải readable và có đủ semantic để embedding retrieval work
- Point ID `digest:{run_id}` — nếu run chạy lại với cùng run_id → upsert overwrite point cũ (expected behavior)
- `IndexDigest` không dùng PostgreSQL tracking — tương tự `IndexInsight`
- `toIndexDigestInput` có logic convert nullable `SeverityMix` — kiểm tra nil handling đúng
- Tất cả 3 consumer groups phải được close trong `Close()` method (đã implement ở M3)
