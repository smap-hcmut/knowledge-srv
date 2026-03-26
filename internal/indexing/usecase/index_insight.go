package usecase

import (
	"context"
	"crypto/sha256"
	"fmt"
	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/indexing"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
	"time"
)

type insightCardPayload struct {
	ProjectID           string                 `json:"project_id"`
	CampaignID          string                 `json:"campaign_id"`
	RunID               string                 `json:"run_id"`
	RAGDocumentType     string                 `json:"rag_document_type"`
	InsightType         string                 `json:"insight_type"`
	Title               string                 `json:"title"`
	Summary             string                 `json:"summary"`
	Confidence          float64                `json:"confidence"`
	AnalysisWindowStart string                 `json:"analysis_window_start"`
	AnalysisWindowEnd   string                 `json:"analysis_window_end"`
	SupportingMetrics   map[string]interface{} `json:"supporting_metrics,omitempty"`
	EvidenceReferences  []string               `json:"evidence_references,omitempty"`
}

func (uc *implUseCase) IndexInsight(ctx context.Context, input indexing.IndexInsightInput) (indexing.IndexInsightOutput, error) {
	startTime := time.Now()

	if input.Title == "" {
		return indexing.IndexInsightOutput{}, indexing.ErrInsightTitleEmpty
	}
	if input.Summary == "" {
		return indexing.IndexInsightOutput{}, indexing.ErrInsightSummaryEmpty
	}

	embedText := input.Title + ". " + input.Summary

	genOutput, err := uc.embeddingUC.Generate(ctx, embedding.GenerateInput{Text: embedText})
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.IndexInsight: embedding failed: %v", err)
		return indexing.IndexInsightOutput{}, fmt.Errorf("%w: %v", indexing.ErrEmbeddingFailed, err)
	}

	hash := sha256.Sum256([]byte(input.Title))
	pointID := fmt.Sprintf("insight:%s:%s:%x", input.RunID, input.InsightType, hash[:6])

	payload := insightCardPayload{
		ProjectID:           input.ProjectID,
		CampaignID:          input.CampaignID,
		RunID:               input.RunID,
		RAGDocumentType:     "insight_card",
		InsightType:         input.InsightType,
		Title:               input.Title,
		Summary:             input.Summary,
		Confidence:          input.Confidence,
		AnalysisWindowStart: input.AnalysisWindowStart,
		AnalysisWindowEnd:   input.AnalysisWindowEnd,
		SupportingMetrics:   input.SupportingMetrics,
		EvidenceReferences:  input.EvidenceReferences,
	}

	if err := uc.pointUC.EnsureCollection(ctx, point.CollectionMacroInsights, defaultVectorSize); err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.IndexInsight: failed to ensure collection %s: %v", point.CollectionMacroInsights, err)
		return indexing.IndexInsightOutput{}, err
	}

	err = uc.pointUC.Upsert(ctx, point.UpsertInput{
		CollectionName: point.CollectionMacroInsights,
		Points: []model.Point{
			{
				ID:      pointID,
				Vector:  genOutput.Vector,
				Payload: uc.payloadFromStruct(payload),
			},
		},
	})
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.IndexInsight: qdrant upsert failed: %v", err)
		return indexing.IndexInsightOutput{}, fmt.Errorf("%w: %v", indexing.ErrQdrantUpsertFailed, err)
	}

	output := indexing.IndexInsightOutput{
		PointID:  pointID,
		Duration: time.Since(startTime),
	}

	uc.l.Infof(ctx, "indexing.usecase.IndexInsight: indexed insight %s (point: %s, duration: %s)",
		input.InsightType, output.PointID, output.Duration)

	return output, nil
}
