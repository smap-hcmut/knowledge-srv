package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// generateContentHash produces a stable per-content fingerprint used both for
// the indexed_documents.content_hash column and the pre-embed dedup check.
// SHA-256 keyed only by the normalized text — any caller-provided salt would
// break dedup across consumers.
func (uc *implUseCase) generateContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

const (
	// maxPayloadContentChars caps the content snippet stored in Qdrant. Long
	// transcripts (TikTok captions, multi-paragraph reviews) easily push a
	// single point past the 64MB serialized limit when payload also carries
	// summaries, aspects and source URLs; the full text remains addressable
	// in Postgres via SourceID + RootID.
	maxPayloadContentChars = 800

	// maxPayloadSummaryChars is the soft cap for the summary fallback when
	// an upstream summary is missing — shorter than the raw content snippet
	// because the summary is meant to be a TL;DR, not the full quote.
	maxPayloadSummaryChars = 400
)

// trimPayloadText cuts s at most max runes long and appends an ellipsis so
// callers can tell the snippet was truncated without re-fetching the full
// document.
func trimPayloadText(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

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

// maxVoyageBatch caps how many texts we hand to Voyage in one call. Voyage's
// API enforces 128; the spare 64 leaves headroom in case a future model
// tightens the limit and keeps each request small enough that a single 429
// retry does not pause the whole batch for too long.
const maxVoyageBatch = 64

// pendingIndex represents one document that has passed validation/dedup and
// is waiting on a batched Voyage call. Holds everything the upsert phase
// needs so we do not rebuild it after the embedding round-trip.
type pendingIndex struct {
	doc            indexing.InsightMessageInput
	analyticsID    string
	sourceID       string
	pointID        string
	contentHash    string
	embeddingText  string
	cleanText      string
	startTime      time.Time
}

func (uc *implUseCase) processInsightBatch(ctx context.Context, input indexing.IndexBatchInput) indexing.IndexBatchOutput {
	collectionName := fmt.Sprintf("proj_%s", input.ProjectID)

	// Phase 1 — validate + dedup each document in parallel. The output is
	// either a "pending" record ready for batched embedding, or one of the
	// terminal STATUS_SKIPPED / STATUS_FAILED counts.
	type prepareResult struct {
		pending *pendingIndex
		status  string // "" when pending is set
	}
	results := make([]prepareResult, len(input.Documents))
	var prepWG sync.WaitGroup
	sem := make(chan struct{}, indexing.MaxConcurrency)
	for i := range input.Documents {
		i := i
		prepWG.Add(1)
		sem <- struct{}{}
		go func() {
			defer prepWG.Done()
			defer func() { <-sem }()
			pending, status := uc.prepareIndex(ctx, input.ProjectID, input.CampaignID, input.Documents[i])
			results[i] = prepareResult{pending: pending, status: status}
		}()
	}
	prepWG.Wait()

	var (
		indexed int
		failed  int
		skipped int
		mu      sync.Mutex
	)

	pendings := make([]*pendingIndex, 0, len(results))
	for _, r := range results {
		switch {
		case r.pending != nil:
			pendings = append(pendings, r.pending)
		case r.status == indexing.STATUS_SKIPPED:
			skipped++
		case r.status == indexing.STATUS_FAILED:
			failed++
		}
	}

	// Phase 2 — batch the embedding round-trips so a 200-document Kafka
	// payload turns into a handful of Voyage calls instead of 200 separate
	// API requests. Embedding still goes through the cache first so already
	// fingerprinted texts return without touching Voyage at all.
	for start := 0; start < len(pendings); start += maxVoyageBatch {
		end := start + maxVoyageBatch
		if end > len(pendings) {
			end = len(pendings)
		}
		chunk := pendings[start:end]

		texts := make([]string, len(chunk))
		for j, p := range chunk {
			texts[j] = p.embeddingText
		}
		embeddingStart := time.Now()
		genOut, err := uc.embeddingUC.GenerateMany(ctx, embedding.GenerateManyInput{Texts: texts})
		embeddingTime := int(time.Since(embeddingStart).Milliseconds())
		if err != nil {
			uc.l.Errorf(ctx, "indexing.usecase.processInsightBatch: batch embedding failed for project %s (chunk %d-%d): %v", input.ProjectID, start, end-1, err)
			failed += len(chunk)
			continue
		}
		if len(genOut.Vectors) != len(chunk) {
			uc.l.Errorf(ctx, "indexing.usecase.processInsightBatch: vector count mismatch (expected %d, got %d) for project %s", len(chunk), len(genOut.Vectors), input.ProjectID)
			failed += len(chunk)
			continue
		}

		// Phase 3 — fan-out the upsert step concurrently per item; each
		// upsert is independent so we hold MaxConcurrency goroutines in
		// flight while the next embedding chunk warms up.
		g, gctx := errgroup.WithContext(ctx)
		g.SetLimit(indexing.MaxConcurrency)
		for idx := range chunk {
			idx := idx
			pending := chunk[idx]
			vector := genOut.Vectors[idx]
			g.Go(func() error {
				status := uc.finalizeIndex(gctx, input.ProjectID, input.CampaignID, collectionName, pending, vector, embeddingTime)
				mu.Lock()
				defer mu.Unlock()
				switch status {
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
	}

	return indexing.IndexBatchOutput{
		ProjectID: input.ProjectID,
		Indexed:   indexed,
		Skipped:   skipped,
		Failed:    failed,
	}
}

// prepareIndex runs validation + dedup for one document. Returns either a
// pendingIndex record ready for batched embedding, or a terminal status that
// the caller increments directly.
func (uc *implUseCase) prepareIndex(
	ctx context.Context,
	projectID string,
	campaignID string,
	doc indexing.InsightMessageInput,
) (*pendingIndex, string) {
	startTime := time.Now()

	if !doc.RAG {
		if !isSocialPlatformFallback(doc.Identity.Platform) {
			uc.l.Warnf(ctx, "indexing.usecase.prepareIndex: skipped doc %s for project %s: gate=rag_disabled", doc.Identity.UapID, projectID)
			return nil, indexing.STATUS_SKIPPED
		}
		uc.l.Warnf(ctx, "indexing.usecase.prepareIndex: doc %s for project %s: rag_disabled_fallback_platform=%s", doc.Identity.UapID, projectID, doc.Identity.Platform)
	}

	cleanText := strings.TrimSpace(doc.Content.CleanText)
	if doc.Identity.UapID == "" || cleanText == "" {
		uc.l.Warnf(ctx, "indexing.usecase.prepareIndex: skipped doc for project %s: gate=missing_required_fields", projectID)
		return nil, indexing.STATUS_SKIPPED
	}

	if shouldIndex, reason := shouldIndexInsight(doc, cleanText); !shouldIndex {
		uc.l.Warnf(ctx, "indexing.usecase.prepareIndex: skipped doc %s for project %s campaign %s: gate=%s", doc.Identity.UapID, projectID, campaignID, reason)
		return nil, indexing.STATUS_SKIPPED
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

	if existing, err := uc.postgreRepo.GetOneDocument(ctx, repo.GetOneDocumentOptions{AnalyticsID: analyticsID}); err == nil &&
		existing.ID != "" &&
		existing.Status == indexing.STATUS_INDEXED &&
		existing.ContentHash == contentHash {
		uc.l.Debugf(ctx, "indexing.usecase.prepareIndex: dedup skip uap=%s analytics_id=%s", doc.Identity.UapID, analyticsID)
		return nil, indexing.STATUS_SKIPPED
	}

	return &pendingIndex{
		doc:           doc,
		analyticsID:   analyticsID,
		sourceID:      sourceID,
		pointID:       pointID,
		contentHash:   contentHash,
		embeddingText: buildEmbeddingText(doc, cleanText),
		cleanText:     cleanText,
		startTime:     startTime,
	}, ""
}

// finalizeIndex takes the embedded vector and runs Qdrant upsert + Postgres
// metadata write for one pending document. Split out so the batched embedding
// phase can hand vectors back to MaxConcurrency goroutines.
func (uc *implUseCase) finalizeIndex(
	ctx context.Context,
	projectID string,
	campaignID string,
	collectionName string,
	pending *pendingIndex,
	vector []float32,
	embeddingTime int,
) string {
	payload := uc.buildInsightPayload(pending.analyticsID, pending.sourceID, pending.pointID, projectID, campaignID, pending.doc)

	upsertStart := time.Now()
	if err := uc.pointUC.Upsert(ctx, point.UpsertInput{
		CollectionName: collectionName,
		Points: []model.Point{
			{
				ID:      pending.pointID,
				Vector:  vector,
				Payload: payload,
			},
		},
	}); err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.finalizeIndex: qdrant upsert failed for %s: %v", pending.doc.Identity.UapID, err)
		return indexing.STATUS_FAILED
	}
	upsertTime := int(time.Since(upsertStart).Milliseconds())

	now := time.Now()
	_, err := uc.postgreRepo.UpsertDocument(ctx, repo.UpsertDocumentOptions{
		AnalyticsID:     pending.analyticsID,
		ProjectID:       projectID,
		SourceID:        pending.sourceID,
		QdrantPointID:   pending.pointID,
		CollectionName:  collectionName,
		ContentHash:     pending.contentHash,
		Status:          indexing.STATUS_INDEXED,
		RetryCount:      0,
		EmbeddingTimeMs: embeddingTime,
		UpsertTimeMs:    upsertTime,
		TotalTimeMs:     int(time.Since(pending.startTime).Milliseconds()),
		IndexedAt:       &now,
	})
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.finalizeIndex: metadata upsert failed for %s: %v", pending.doc.Identity.UapID, err)
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
		// Content kept as a short snippet so Qdrant payload stays well below
		// the 64MB point limit. Full text lives in Postgres
		// (analytics.post_insight.content); search results carry RootID +
		// SourceID + Permalink so callers can hydrate the original document
		// when they actually need it.
		Content:           trimPayloadText(cleanText, maxPayloadContentChars),
		ContentSummary:    firstNonEmptyString(doc.Content.Summary, trimPayloadText(cleanText, maxPayloadSummaryChars)),
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
