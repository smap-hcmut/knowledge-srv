package usecase

import (
	"context"
	"fmt"
	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/indexing"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
	"strings"
	"time"
)

type digestPayload struct {
	ProjectID           string            `json:"project_id"`
	CampaignID          string            `json:"campaign_id"`
	RunID               string            `json:"run_id"`
	RAGDocumentType     string            `json:"rag_document_type"`
	AnalysisWindowStart string            `json:"analysis_window_start"`
	AnalysisWindowEnd   string            `json:"analysis_window_end"`
	DomainOverlay       string            `json:"domain_overlay"`
	Platform            string            `json:"platform"`
	TotalMentions       int               `json:"total_mentions"`
	TopEntities         []digestTopEntity `json:"top_entities,omitempty"`
	TopTopics           []digestTopTopic  `json:"top_topics,omitempty"`
	TopIssues           []digestTopIssue  `json:"top_issues,omitempty"`
}

type digestTopEntity struct {
	CanonicalEntityID string  `json:"canonical_entity_id"`
	EntityName        string  `json:"entity_name"`
	EntityType        string  `json:"entity_type"`
	MentionCount      int     `json:"mention_count"`
	MentionShare      float64 `json:"mention_share"`
}

type digestTopTopic struct {
	TopicKey            string   `json:"topic_key"`
	TopicLabel          string   `json:"topic_label"`
	MentionCount        int      `json:"mention_count"`
	MentionShare        float64  `json:"mention_share"`
	BuzzScoreProxy      *float64 `json:"buzz_score_proxy,omitempty"`
	QualityScore        *float64 `json:"quality_score,omitempty"`
	RepresentativeTexts []string `json:"representative_texts,omitempty"`
}

type digestTopIssue struct {
	IssueCategory      string             `json:"issue_category"`
	MentionCount       int                `json:"mention_count"`
	IssuePressureProxy float64            `json:"issue_pressure_proxy"`
	SeverityMix        *digestSeverityMix `json:"severity_mix,omitempty"`
}

type digestSeverityMix struct {
	Low    float64 `json:"low"`
	Medium float64 `json:"medium"`
	High   float64 `json:"high"`
}

func (uc *implUseCase) IndexDigest(ctx context.Context, input indexing.IndexDigestInput) (indexing.IndexDigestOutput, error) {
	startTime := time.Now()

	prose := uc.buildDigestProse(input)
	if prose == "" {
		return indexing.IndexDigestOutput{}, indexing.ErrDigestBuildFailed
	}

	genOutput, err := uc.embeddingUC.Generate(ctx, embedding.GenerateInput{Text: prose})
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.IndexDigest: embedding failed: %v", err)
		return indexing.IndexDigestOutput{}, fmt.Errorf("%w: %v", indexing.ErrEmbeddingFailed, err)
	}

	pointID := fmt.Sprintf("digest:%s", input.RunID)

	payload := digestPayload{
		ProjectID:           input.ProjectID,
		CampaignID:          input.CampaignID,
		RunID:               input.RunID,
		RAGDocumentType:     "report_digest",
		AnalysisWindowStart: input.AnalysisWindowStart,
		AnalysisWindowEnd:   input.AnalysisWindowEnd,
		DomainOverlay:       input.DomainOverlay,
		Platform:            input.Platform,
		TotalMentions:       input.TotalMentions,
		TopEntities:         mapDigestTopEntities(input.TopEntities),
		TopTopics:           mapDigestTopTopics(input.TopTopics),
		TopIssues:           mapDigestTopIssues(input.TopIssues),
	}

	if err := uc.pointUC.EnsureCollection(ctx, point.CollectionMacroInsights, defaultVectorSize); err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.IndexDigest: failed to ensure collection %s: %v", point.CollectionMacroInsights, err)
		return indexing.IndexDigestOutput{}, err
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
		uc.l.Errorf(ctx, "indexing.usecase.IndexDigest: qdrant upsert failed: %v", err)
		return indexing.IndexDigestOutput{}, fmt.Errorf("%w: %v", indexing.ErrQdrantUpsertFailed, err)
	}

	output := indexing.IndexDigestOutput{
		PointID:  pointID,
		Duration: time.Since(startTime),
	}

	uc.l.Infof(ctx, "indexing.usecase.IndexDigest: indexed digest for run %s (point: %s, duration: %s)",
		input.RunID, output.PointID, output.Duration)

	return output, nil
}

func (uc *implUseCase) buildDigestProse(input indexing.IndexDigestInput) string {
	if input.DomainOverlay == "" && input.Platform == "" && input.TotalMentions == 0 && len(input.TopEntities) == 0 && len(input.TopTopics) == 0 && len(input.TopIssues) == 0 {
		return ""
	}

	var b strings.Builder

	fmt.Fprintf(&b, "Campaign Report: %s\n", input.DomainOverlay)
	fmt.Fprintf(&b, "Platform: %s | Total Mentions: %d\n", input.Platform, input.TotalMentions)
	fmt.Fprintf(&b, "Analysis Window: %s to %s\n\n", input.AnalysisWindowStart, input.AnalysisWindowEnd)

	if len(input.TopEntities) > 0 {
		b.WriteString("Top Brands:\n")
		limit := len(input.TopEntities)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			e := input.TopEntities[i]
			fmt.Fprintf(&b, "- %s: %d mentions (%.1f%% share)\n", e.EntityName, e.MentionCount, e.MentionShare*100)
		}
		b.WriteString("\n")
	}

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

	if len(input.TopIssues) > 0 {
		b.WriteString("Critical Issues:\n")
		limit := len(input.TopIssues)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			issue := input.TopIssues[i]
			fmt.Fprintf(&b, "- %s: %d mentions (pressure: %.2f)\n", issue.IssueCategory, issue.MentionCount, issue.IssuePressureProxy)
		}
	}

	return strings.TrimSpace(b.String())
}

func mapDigestTopEntities(in []indexing.TopEntityInput) []digestTopEntity {
	if len(in) == 0 {
		return nil
	}
	out := make([]digestTopEntity, len(in))
	for i, e := range in {
		out[i] = digestTopEntity{
			CanonicalEntityID: e.CanonicalEntityID,
			EntityName:        e.EntityName,
			EntityType:        e.EntityType,
			MentionCount:      e.MentionCount,
			MentionShare:      e.MentionShare,
		}
	}
	return out
}

func mapDigestTopTopics(in []indexing.TopTopicInput) []digestTopTopic {
	if len(in) == 0 {
		return nil
	}
	out := make([]digestTopTopic, len(in))
	for i, t := range in {
		out[i] = digestTopTopic{
			TopicKey:            t.TopicKey,
			TopicLabel:          t.TopicLabel,
			MentionCount:        t.MentionCount,
			MentionShare:        t.MentionShare,
			BuzzScoreProxy:      t.BuzzScoreProxy,
			QualityScore:        t.QualityScore,
			RepresentativeTexts: t.RepresentativeTexts,
		}
	}
	return out
}

func mapDigestTopIssues(in []indexing.TopIssueInput) []digestTopIssue {
	if len(in) == 0 {
		return nil
	}
	out := make([]digestTopIssue, len(in))
	for i, iss := range in {
		var severityMix *digestSeverityMix
		if iss.SeverityMix != nil {
			severityMix = &digestSeverityMix{
				Low:    iss.SeverityMix.Low,
				Medium: iss.SeverityMix.Medium,
				High:   iss.SeverityMix.High,
			}
		}
		out[i] = digestTopIssue{
			IssueCategory:      iss.IssueCategory,
			MentionCount:       iss.MentionCount,
			IssuePressureProxy: iss.IssuePressureProxy,
			SeverityMix:        severityMix,
		}
	}
	return out
}
