package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"knowledge-srv/internal/indexing/delivery/kafka"

	"github.com/IBM/sarama"
	"github.com/smap-hcmut/shared-libs/go/auth"
)

// handleBatchCompletedMessage receives Layer 3 message and routes by format.
func (c *consumer) handleBatchCompletedMessage(msg *sarama.ConsumerMessage) error {
	ctx := context.Background()

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: Processing message from partition %d, offset %d",
		msg.Partition, msg.Offset)

	// 1. Try new format first (documents[])
	var newMsg kafka.BatchCompletedMessage
	if err := json.Unmarshal(msg.Value, &newMsg); err == nil && len(newMsg.Documents) > 0 {
		c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: Detected new format (documents[]), project=%s docs=%d",
			newMsg.ProjectID, len(newMsg.Documents))
		return c.handleNewBatchCompleted(ctx, newMsg)
	}

	// 2. Fallback to legacy format (file_url)
	var legacyMsg kafka.LegacyBatchCompletedMessage
	if err := json.Unmarshal(msg.Value, &legacyMsg); err == nil && legacyMsg.FileURL != "" {
		c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: Detected legacy format (file_url), batch=%s",
			legacyMsg.BatchID)
		return c.handleLegacyBatchCompleted(ctx, legacyMsg)
	}

	// 3. Invalid message format
	c.l.Warnf(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: Cannot parse message (neither new nor legacy format), skipping")
	return nil
}

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

func (c *consumer) handleInsightsPublishedMessage(msg *sarama.ConsumerMessage) error {
	ctx := context.Background()

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleInsightsPublishedMessage: Processing message from partition %d, offset %d",
		msg.Partition, msg.Offset)

	var message kafka.InsightsPublishedMessage
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		c.l.Warnf(ctx, "indexing.delivery.kafka.consumer.handleInsightsPublishedMessage: Invalid message format (skipping): %v", err)
		return nil
	}

	if !message.ShouldIndex {
		c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleInsightsPublishedMessage: should_index=false, skipping insight %s/%s",
			message.RunID, message.InsightType)
		return nil
	}

	if message.ProjectID == "" || message.RunID == "" || message.InsightType == "" || message.Title == "" {
		c.l.Warnf(ctx, "indexing.delivery.kafka.consumer.handleInsightsPublishedMessage: Missing required fields (skipping)")
		return nil
	}

	input := toIndexInsightInput(message)

	sc := auth.Scope{UserID: "system", Role: "system"}
	ctx = auth.SetScopeToContext(ctx, sc)

	output, err := c.uc.IndexInsight(ctx, input)
	if err != nil {
		c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.handleInsightsPublishedMessage: IndexInsight failed: %v", err)
		return fmt.Errorf("usecase error: %w", err)
	}

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleInsightsPublishedMessage: Indexed insight %s (point: %s, duration: %s)",
		message.InsightType, output.PointID, output.Duration)
	return nil
}

func (c *consumer) handleReportDigestMessage(msg *sarama.ConsumerMessage) error {
	ctx := context.Background()

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleReportDigestMessage: Processing message from partition %d, offset %d",
		msg.Partition, msg.Offset)

	var message kafka.ReportDigestMessage
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		c.l.Warnf(ctx, "indexing.delivery.kafka.consumer.handleReportDigestMessage: Invalid message format (skipping): %v", err)
		return nil
	}

	if !message.ShouldIndex {
		c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleReportDigestMessage: should_index=false, skipping run %s", message.RunID)
		return nil
	}

	if message.ProjectID == "" || message.RunID == "" || message.DomainOverlay == "" {
		c.l.Warnf(ctx, "indexing.delivery.kafka.consumer.handleReportDigestMessage: Missing required fields (skipping)")
		return nil
	}

	input := toIndexDigestInput(message)

	sc := auth.Scope{UserID: "system", Role: "system"}
	ctx = auth.SetScopeToContext(ctx, sc)

	output, err := c.uc.IndexDigest(ctx, input)
	if err != nil {
		c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.handleReportDigestMessage: IndexDigest failed: %v", err)
		return fmt.Errorf("usecase error: %w", err)
	}

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleReportDigestMessage: Indexed digest run %s (point: %s, duration: %s)",
		message.RunID, output.PointID, output.Duration)

	return nil
}
