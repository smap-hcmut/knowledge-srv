package usecase

import (
	"context"
	"fmt"
	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/indexing"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// IndexBatch indexes a batch of direct payload documents from Kafka.
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

	collectionName := fmt.Sprintf("proj_%s", input.ProjectID)
	if err := uc.pointUC.EnsureCollection(ctx, collectionName, defaultVectorSize); err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.IndexBatch: failed to ensure collection %s: %v", collectionName, err)
		return indexing.IndexBatchOutput{}, err
	}

	result := uc.processInsightBatch(ctx, input)
	result.TotalRecords = len(input.Documents)
	result.Duration = time.Since(startTime)

	if result.Indexed > 0 {
		if err := uc.cacheRepo.InvalidateSearchCache(ctx, input.ProjectID); err != nil {
			uc.l.Warnf(ctx, "indexing.usecase.IndexBatch: Failed to invalidate cache: %v", err)
		}
	}

	uc.l.Infof(ctx, "indexing.usecase.IndexBatch: project=%s total=%d indexed=%d skipped=%d failed=%d duration=%s",
		input.ProjectID, result.TotalRecords, result.Indexed, result.Skipped, result.Failed, result.Duration)

	return result, nil
}

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
			case indexing.STATUS_INDEXED:
				indexed++
			case indexing.STATUS_SKIPPED:
				skipped++
			case indexing.STATUS_FAILED:
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

func (uc *implUseCase) indexSingleInsight(
	ctx context.Context,
	projectID string,
	campaignID string,
	collectionName string,
	doc indexing.InsightMessageInput,
) string {
	if !doc.RAG {
		return indexing.STATUS_SKIPPED
	}

	cleanText := strings.TrimSpace(doc.Content.CleanText)
	if doc.Identity.UapID == "" || cleanText == "" {
		uc.l.Warnf(ctx, "indexing.usecase.indexSingleInsight: skipping doc with empty uap_id or clean_text")
		return indexing.STATUS_SKIPPED
	}
	if !isIndexableInsight(doc, cleanText) {
		return indexing.STATUS_SKIPPED
	}

	embeddingText := buildEmbeddingText(doc, cleanText)
	genOutput, err := uc.embeddingUC.Generate(ctx, embedding.GenerateInput{Text: embeddingText})
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.indexSingleInsight: embedding failed for %s: %v", doc.Identity.UapID, err)
		return indexing.STATUS_FAILED
	}

	payload := uc.buildInsightPayload(projectID, campaignID, doc)

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
		return indexing.STATUS_FAILED
	}

	return indexing.STATUS_INDEXED
}

func (uc *implUseCase) buildInsightPayload(
	projectID string,
	campaignID string,
	doc indexing.InsightMessageInput,
) map[string]interface{} {
	cleanText := strings.TrimSpace(doc.Content.CleanText)
	payload := insightPayload{
		ProjectID:        projectID,
		CampaignID:       campaignID,
		UapID:            doc.Identity.UapID,
		UapType:          doc.Identity.UapType,
		UapMediaType:     doc.Identity.UapMediaType,
		Platform:         doc.Identity.Platform,
		PublishedAt:      doc.Identity.PublishedAt,
		Content:          cleanText,
		ContentSummary:   firstNonEmptyString(doc.Content.Summary, cleanText),
		ContextSummary:   strings.TrimSpace(doc.Content.ContextSummary),
		SentimentLabel:   doc.NLP.Sentiment.Label,
		SentimentScore:   doc.NLP.Sentiment.Score,
		Aspects:          mapInsightAspects(doc.NLP.Aspects),
		Entities:         mapInsightEntities(doc.NLP.Entities),
		ImpactScore:      doc.Business.Impact.ImpactScore,
		RelevanceScore:   doc.Business.RelevanceScore,
		RelevanceReasons: doc.Business.RelevanceReasons,
		Priority:         doc.Business.Impact.Priority,
		Likes:            doc.Business.Impact.Engagement.Likes,
		Comments:         doc.Business.Impact.Engagement.Comments,
		Shares:           doc.Business.Impact.Engagement.Shares,
		Views:            doc.Business.Impact.Engagement.Views,
	}

	return uc.payloadFromStruct(payload)
}

func isIndexableInsight(doc indexing.InsightMessageInput, cleanText string) bool {
	if len([]rune(cleanText)) < indexing.MinContentLength {
		return false
	}
	if len([]rune(cleanText)) < 20 {
		return false
	}
	if businessRelevanceScore(doc, cleanText) < indexing.MinBusinessRelevanceScore {
		return false
	}
	if hasInsightSignal(doc) {
		return true
	}
	return containsBusinessSignal(cleanText)
}

func hasInsightSignal(doc indexing.InsightMessageInput) bool {
	if len(doc.NLP.Aspects) > 0 || len(doc.NLP.Entities) > 0 {
		return true
	}
	if doc.Business.Impact.ImpactScore >= 0.15 {
		return true
	}
	priority := strings.ToUpper(strings.TrimSpace(doc.Business.Impact.Priority))
	if priority == "HIGH" || priority == "MEDIUM" || priority == "CRITICAL" {
		return true
	}
	return false
}

func buildEmbeddingText(doc indexing.InsightMessageInput, cleanText string) string {
	contextSummary := strings.TrimSpace(doc.Content.ContextSummary)
	if contextSummary == "" {
		return cleanText
	}
	return cleanText + "\n\nContext: " + contextSummary
}

func businessRelevanceScore(doc indexing.InsightMessageInput, cleanText string) float64 {
	if doc.Business.RelevanceScore > 0 {
		return doc.Business.RelevanceScore
	}
	if containsBusinessSignal(cleanText) {
		return 0.45
	}
	if containsBusinessSignal(doc.Content.ContextSummary) && len([]rune(cleanText)) >= 35 {
		return 0.36
	}
	return 0.0
}

func containsBusinessSignal(text string) bool {
	lowered := strings.ToLower(text)
	signals := []string{
		"ahamove",
		"aha move",
		"ahatruck",
		"giao hang",
		"giao hàng",
		"ship",
		"shipper",
		"tai xe",
		"tài xế",
		"don hang",
		"đơn hàng",
		"cod",
		"thu ho",
		"thu hộ",
		"huy don",
		"hủy đơn",
		"tong dai",
		"tổng đài",
		"ho tro",
		"hỗ trợ",
	}
	for _, signal := range signals {
		if strings.Contains(lowered, signal) {
			return true
		}
	}
	return false
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
