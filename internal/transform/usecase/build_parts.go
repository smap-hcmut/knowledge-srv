package usecase

import (
	"context"
	"crypto/sha256"
	"fmt"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/notebook"
	"knowledge-srv/internal/point"
	"knowledge-srv/internal/transform"
	"strings"
	"time"

	pb "github.com/qdrant/go-client/qdrant"
)

const maxMarkdownBytes = 500_000

func (uc *implUseCase) BuildParts(ctx context.Context, input transform.TransformInput) ([]notebook.MarkdownPart, error) {
	if input.ProjectID == "" || input.RunID == "" {
		return nil, nil
	}

	periodLabel, err := isoWeekLabel(input.WindowStart)
	if err != nil {
		periodLabel = isoWeekLabelNow()
	}

	digest, err := uc.scrollDigestSection(ctx, input.RunID)
	if err != nil {
		return nil, err
	}
	insights, err := uc.scrollInsightSection(ctx, input.RunID)
	if err != nil {
		return nil, err
	}
	posts, nPosts, err := uc.scrollHighImpactSection(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(strings.Join([]string{digest, insights, posts}, "\n\n"))
	if content == "" {
		return nil, nil
	}

	return chunkIntoParts(content, input.CampaignID, periodLabel, nPosts), nil
}

func isoWeekLabel(s string) (string, error) {
	if s == "" {
		return "", fmt.Errorf("empty window")
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return "", err
	}
	y, w := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w), nil
}

func isoWeekLabelNow() string {
	t := time.Now().UTC()
	y, w := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

func (uc *implUseCase) scrollDigestSection(ctx context.Context, runID string) (string, error) {
	filter := macroRunTypeFilter(runID, "report_digest")
	points, err := uc.pointUC.Scroll(ctx, point.ScrollInput{
		CollectionName: point.CollectionMacroInsights,
		Filter:         filter,
		Limit:          10,
		WithPayload:    true,
	})
	if err != nil {
		return "", err
	}
	if len(points) == 0 {
		return "", nil
	}
	return "## Campaign Report\n\n" + formatDigestFromPayload(points[0].Payload), nil
}

func (uc *implUseCase) scrollInsightSection(ctx context.Context, runID string) (string, error) {
	filter := macroRunTypeFilter(runID, "insight_card")
	points, err := uc.pointUC.Scroll(ctx, point.ScrollInput{
		CollectionName: point.CollectionMacroInsights,
		Filter:         filter,
		Limit:          200,
		WithPayload:    true,
	})
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## Insights for Run %s\n\n", runID)
	if len(points) == 0 {
		b.WriteString("(no insight cards indexed)\n")
		return b.String(), nil
	}
	for _, p := range points {
		b.WriteString(formatInsightCard(p.Payload))
		b.WriteString("\n")
	}
	return b.String(), nil
}

func (uc *implUseCase) scrollHighImpactSection(ctx context.Context, projectID string) (string, int, error) {
	// priority == HIGH OR impact_score > 60
	gt := 60.0
	filter := &pb.Filter{
		Should: []*pb.Condition{
			{
				ConditionOneOf: &pb.Condition_Field{
					Field: &pb.FieldCondition{
						Key: "priority",
						Match: &pb.Match{
							MatchValue: &pb.Match_Keyword{Keyword: "HIGH"},
						},
					},
				},
			},
			{
				ConditionOneOf: &pb.Condition_Field{
					Field: &pb.FieldCondition{
						Key: "impact_score",
						Range: &pb.Range{
							Gt: &gt,
						},
					},
				},
			},
		},
	}
	col := fmt.Sprintf("proj_%s", projectID)
	points, err := uc.pointUC.Scroll(ctx, point.ScrollInput{
		CollectionName: col,
		Filter:         filter,
		Limit:          uint64(uc.maxPostsPerPart),
		WithPayload:    true,
	})
	if err != nil {
		uc.l.Warnf(ctx, "transform.scrollHighImpactSection: collection %s: %v", col, err)
		return fmt.Sprintf("## High-Impact Posts\n\n(collection not available or empty)\n"), 0, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## High-Impact Posts\n\n")
	if len(points) == 0 {
		b.WriteString("(no matching posts)\n")
		return b.String(), 0, nil
	}
	for i, p := range points {
		b.WriteString(formatProjectPost(p))
		if i < len(points)-1 {
			b.WriteString("\n---\n\n")
		}
	}
	return b.String(), len(points), nil
}

func macroRunTypeFilter(runID, docType string) *pb.Filter {
	return &pb.Filter{
		Must: []*pb.Condition{
			fieldEqKeyword("run_id", runID),
			fieldEqKeyword("rag_document_type", docType),
		},
	}
}

func fieldEqKeyword(key, keyword string) *pb.Condition {
	return &pb.Condition{
		ConditionOneOf: &pb.Condition_Field{
			Field: &pb.FieldCondition{
				Key: key,
				Match: &pb.Match{
					MatchValue: &pb.Match_Keyword{Keyword: keyword},
				},
			},
		},
	}
}

func chunkIntoParts(full, campaignID, periodLabel string, postCount int) []notebook.MarkdownPart {
	var parts []notebook.MarkdownPart
	remaining := full
	partNum := 1
	totalParts := (len(full) + maxMarkdownBytes - 1) / maxMarkdownBytes
	if totalParts < 1 {
		totalParts = 1
	}
	for len(remaining) > 0 {
		take := maxMarkdownBytes
		if len(remaining) < take {
			take = len(remaining)
		}
		chunk := remaining[:take]
		remaining = remaining[take:]

		sum := sha256.Sum256([]byte(chunk))
		hash := fmt.Sprintf("%x", sum[:16])

		title := fmt.Sprintf("SMAP | %s | %s", campaignID, periodLabel)
		if totalParts > 1 {
			title = fmt.Sprintf("SMAP | %s | %s | Part %d", campaignID, periodLabel, partNum)
		}

		parts = append(parts, notebook.MarkdownPart{
			Title:       title,
			Content:     chunk,
			WeekLabel:   periodLabel,
			PartNum:     partNum,
			PostCount:   postCount,
			ContentHash: hash,
		})
		partNum++
	}
	return parts
}

func formatProjectPost(p model.Point) string {
	pl := p.Payload
	if pl == nil {
		return ""
	}
	id := str(pl["uap_id"])
	platform := str(pl["platform"])
	uapType := str(pl["uap_type"])
	sent := str(pl["sentiment_label"])
	prio := str(pl["priority"])
	impact := num(pl["impact_score"])
	body := str(pl["content_summary"])
	if body == "" {
		body = str(pl["clean_text"])
	}
	likes := intFrom(pl["likes"])
	comments := intFrom(pl["comments"])
	shares := intFrom(pl["shares"])
	views := intFrom(pl["views"])
	var b strings.Builder
	fmt.Fprintf(&b, "**%s** [%s | %s | %s]\n", id, strings.ToUpper(platform), uapType, sent)
	if prio != "" || impact > 0 {
		fmt.Fprintf(&b, "priority=%s | impact=%.2f\n", prio, impact)
	}
	trimmed := strings.TrimSpace(body)
	if len(trimmed) > 800 {
		trimmed = trimmed[:800] + "…"
	}
	b.WriteString(trimmed)
	b.WriteString("\n")
	fmt.Fprintf(&b, "Engagement: %d likes · %d comments · %d shares · %d views\n", likes, comments, shares, views)
	return b.String()
}
