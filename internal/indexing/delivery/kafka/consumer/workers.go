package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"knowledge-srv/internal/indexing/delivery/kafka"

	"github.com/IBM/sarama"
	"github.com/smap-hcmut/shared-libs/go/auth"
)

// handleBatchCompletedMessage receives the Layer 3 documents[] indexing message.
func (c *consumer) handleBatchCompletedMessage(msg *sarama.ConsumerMessage) error {
	ctx := context.Background()

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: Processing message from partition %d, offset %d",
		msg.Partition, msg.Offset)

	var message kafka.BatchCompletedMessage
	if err := json.Unmarshal(msg.Value, &message); err != nil {
		c.l.Warnf(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: Invalid message format (skipping): %v", err)
		return nil
	}

	if message.ProjectID == "" {
		c.l.Warnf(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: Missing project_id (skipping)")
		return nil
	}
	if len(message.Documents) == 0 {
		c.l.Warnf(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: Missing documents[] payload (skipping)")
		return nil
	}

	input := toIndexBatchInput(message)

	sc := auth.Scope{UserID: "system", Role: "system"}
	ctx = auth.SetScopeToContext(ctx, sc)

	output, err := c.uc.IndexBatch(ctx, input)
	if err != nil {
		c.l.Errorf(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: IndexBatch failed: %v", err)
		return fmt.Errorf("usecase error: %w", err)
	}

	c.l.Infof(ctx, "indexing.delivery.kafka.consumer.handleBatchCompletedMessage: project=%s total=%d indexed=%d skipped=%d failed=%d",
		message.ProjectID, output.TotalRecords, output.Indexed, output.Skipped, output.Failed)
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
