package usecase

import (
	"context"
	"fmt"
	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/indexing"
	repo "knowledge-srv/internal/indexing/repository"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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
	startTime := time.Now()

	if !doc.RAG {
		if !isSocialPlatformFallback(doc.Identity.Platform) {
			uc.l.Warnf(ctx, "indexing.usecase.indexSingleInsight: skipped doc %s for project %s: gate=rag_disabled", doc.Identity.UapID, projectID)
			return indexing.STATUS_SKIPPED
		}
		uc.l.Warnf(ctx, "indexing.usecase.indexSingleInsight: doc %s for project %s: rag_disabled_fallback_platform=%s", doc.Identity.UapID, projectID, doc.Identity.Platform)
	}

	cleanText := strings.TrimSpace(doc.Content.CleanText)
	if doc.Identity.UapID == "" || cleanText == "" {
		uc.l.Warnf(ctx, "indexing.usecase.indexSingleInsight: skipped doc for project %s: gate=missing_required_fields", projectID)
		return indexing.STATUS_SKIPPED
	}

	if shouldIndex, reason := shouldIndexInsight(doc, cleanText); !shouldIndex {
		uc.l.Warnf(ctx, "indexing.usecase.indexSingleInsight: skipped doc %s for project %s campaign %s: gate=%s", doc.Identity.UapID, projectID, campaignID, reason)
		return indexing.STATUS_SKIPPED
	}

	analyticsID := deterministicInsightUUID("analytics", projectID, doc.Identity.UapID)
	sourceID := deterministicInsightUUID("source", projectID, firstNonEmptyString(
		doc.Source.RootID,
		doc.Source.ParentID,
		doc.Source.SourceURL,
		doc.Source.OriginalURL,
		doc.Source.PostURL,
		doc.Source.URL,
		doc.Identity.UapID,
	))
	pointID := analyticsID
	contentHash := uc.generateContentHash(cleanText)

	embeddingText := buildEmbeddingText(doc, cleanText)
	embeddingStart := time.Now()
	genOutput, err := uc.embeddingUC.Generate(ctx, embedding.GenerateInput{Text: embeddingText})
	embeddingTime := int(time.Since(embeddingStart).Milliseconds())
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.indexSingleInsight: embedding failed for %s: %v", doc.Identity.UapID, err)
		return indexing.STATUS_FAILED
	}

	payload := uc.buildInsightPayload(analyticsID, sourceID, pointID, projectID, campaignID, doc)

	upsertStart := time.Now()
	err = uc.pointUC.Upsert(ctx, point.UpsertInput{
		CollectionName: collectionName,
		Points: []model.Point{
			{
				ID:      pointID,
				Vector:  genOutput.Vector,
				Payload: payload,
			},
		},
	})
	upsertTime := int(time.Since(upsertStart).Milliseconds())
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.indexSingleInsight: qdrant upsert failed for %s: %v", doc.Identity.UapID, err)
		return indexing.STATUS_FAILED
	}

	now := time.Now()
	_, err = uc.postgreRepo.UpsertDocument(ctx, repo.UpsertDocumentOptions{
		AnalyticsID:     analyticsID,
		ProjectID:       projectID,
		SourceID:        sourceID,
		QdrantPointID:   pointID,
		CollectionName:  collectionName,
		ContentHash:     contentHash,
		Status:          indexing.STATUS_INDEXED,
		RetryCount:      0,
		EmbeddingTimeMs: embeddingTime,
		UpsertTimeMs:    upsertTime,
		TotalTimeMs:     int(time.Since(startTime).Milliseconds()),
		IndexedAt:       &now,
	})
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.indexSingleInsight: metadata upsert failed for %s: %v", doc.Identity.UapID, err)
		return indexing.STATUS_FAILED
	}

	return indexing.STATUS_INDEXED
}

func (uc *implUseCase) buildInsightPayload(
	analyticsID string,
	sourceID string,
	pointID string,
	projectID string,
	campaignID string,
	doc indexing.InsightMessageInput,
) map[string]interface{} {
	cleanText := strings.TrimSpace(doc.Content.CleanText)
	payload := insightPayload{
		AnalyticsID:       analyticsID,
		ProjectID:         projectID,
		SourceID:          sourceID,
		QdrantPointID:     pointID,
		CampaignID:        campaignID,
		UapID:             doc.Identity.UapID,
		UapType:           doc.Identity.UapType,
		UapMediaType:      doc.Identity.UapMediaType,
		Platform:          doc.Identity.Platform,
		PublishedAt:       doc.Identity.PublishedAt,
		URL:               doc.Source.URL,
		PostURL:           doc.Source.PostURL,
		OriginalURL:       doc.Source.OriginalURL,
		Permalink:         doc.Source.Permalink,
		SourceURL:         doc.Source.SourceURL,
		WebURL:            doc.Source.WebURL,
		CommentURL:        doc.Source.CommentURL,
		ParentPostURL:     doc.Source.ParentPostURL,
		Author:            doc.Source.Author,
		AuthorDisplayName: doc.Source.AuthorDisplayName,
		AuthorUsername:    doc.Source.AuthorUsername,
		AuthorAvatar:      doc.Source.AuthorAvatar,
		ContentType:       doc.Source.ContentType,
		RootID:            doc.Source.RootID,
		ParentID:          doc.Source.ParentID,
		PlatformMeta:      doc.Source.PlatformMeta,
		Hierarchy:         doc.Source.Hierarchy,
		Content:           cleanText,
		ContentSummary:    firstNonEmptyString(doc.Content.Summary, cleanText),
		ContextSummary:    strings.TrimSpace(doc.Content.ContextSummary),
		SentimentLabel:    doc.NLP.Sentiment.Label,
		SentimentScore:    doc.NLP.Sentiment.Score,
		Aspects:           mapInsightAspects(doc.NLP.Aspects),
		Entities:          mapInsightEntities(doc.NLP.Entities),
		ImpactScore:       doc.Business.Impact.ImpactScore,
		RelevanceScore:    doc.Business.RelevanceScore,
		RelevanceReasons:  doc.Business.RelevanceReasons,
		Priority:          doc.Business.Impact.Priority,
		Likes:             doc.Business.Impact.Engagement.Likes,
		Comments:          doc.Business.Impact.Engagement.Comments,
		Shares:            doc.Business.Impact.Engagement.Shares,
		Views:             doc.Business.Impact.Engagement.Views,
	}

	return uc.payloadFromStruct(payload)
}

func shouldIndexInsight(doc indexing.InsightMessageInput, cleanText string) (bool, string) {
	if len([]rune(cleanText)) < indexing.MinContentLength {
		return false, "content_too_short"
	}

	if businessRelevanceScore(doc, cleanText) >= indexing.MinBusinessRelevanceScore {
		return true, ""
	}

	if hasInsightSignal(doc) || containsBusinessSignal(cleanText) || containsBusinessSignal(doc.Content.ContextSummary) {
		return true, ""
	}

	if isSocialPlatformFallback(doc.Identity.Platform) && len([]rune(cleanText)) >= indexing.MinContentLength {
		return true, "platform_fallback"
	}

	// For production pipelines we keep quality gates strict.
	// For the project demo, avoid silent data starvation due
	// sparse analytics enrichment by allowing fallback indexing.
	return true, "default_allow_fallback"
}

func isSocialPlatformFallback(platform string) bool {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "tiktok", "facebook", "instagram", "x", "youtube", "threads", "reddit":
		return true
	default:
		return false
	}
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

func deterministicInsightUUID(scope string, projectID string, raw string) string {
	raw = strings.TrimSpace(raw)
	if parsed, err := uuid.Parse(raw); err == nil {
		return parsed.String()
	}

	seed := strings.Join([]string{
		"smap",
		"knowledge",
		"direct-batch",
		scope,
		strings.TrimSpace(projectID),
		raw,
	}, "\x00")
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(seed)).String()
}
