package usecase

import (
	"context"
	"fmt"
	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/indexing"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
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

	if doc.Identity.UapID == "" || doc.Content.CleanText == "" {
		uc.l.Warnf(ctx, "indexing.usecase.indexSingleInsight: skipping doc with empty uap_id or clean_text")
		return indexing.STATUS_SKIPPED
	}

	genOutput, err := uc.embeddingUC.Generate(ctx, embedding.GenerateInput{Text: doc.Content.CleanText})
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
	payload := insightPayload{
		ProjectID:      projectID,
		CampaignID:     campaignID,
		UapID:          doc.Identity.UapID,
		UapType:        doc.Identity.UapType,
		UapMediaType:   doc.Identity.UapMediaType,
		Platform:       doc.Identity.Platform,
		PublishedAt:    doc.Identity.PublishedAt,
		ContentSummary: doc.Content.Summary,
		SentimentLabel: doc.NLP.Sentiment.Label,
		SentimentScore: doc.NLP.Sentiment.Score,
		Aspects:        mapInsightAspects(doc.NLP.Aspects),
		Entities:       mapInsightEntities(doc.NLP.Entities),
		ImpactScore:    doc.Business.Impact.ImpactScore,
		Priority:       doc.Business.Impact.Priority,
		Likes:          doc.Business.Impact.Engagement.Likes,
		Comments:       doc.Business.Impact.Engagement.Comments,
		Shares:         doc.Business.Impact.Engagement.Shares,
		Views:          doc.Business.Impact.Engagement.Views,
	}

	return uc.payloadFromStruct(payload)
}
