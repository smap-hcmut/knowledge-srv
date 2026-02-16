package usecase

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"knowledge-srv/internal/indexing"
	repo "knowledge-srv/internal/indexing/repository"
	"knowledge-srv/internal/model"
	"knowledge-srv/pkg/minio"
)

// Index - Index a batch of analytics posts from MinIO file
func (uc *implUseCase) Index(ctx context.Context, input indexing.IndexInput) (indexing.IndexOutput, error) {
	startTime := time.Now()

	// Step 1: Parse file URL to get bucket and object name
	bucket, objectName, err := uc.parseMinIOURL(input.FileURL)
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.Index: Failed to parse MinIO URL: %v", err)
		return indexing.IndexOutput{}, indexing.ErrFileNotFound
	}

	// Step 2: Download file từ MinIO
	reader, _, err := uc.minio.DownloadFile(ctx, &minio.DownloadRequest{
		BucketName: bucket,
		ObjectName: objectName,
	})
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.Index: Failed to download file: %v", err)
		return indexing.IndexOutput{}, indexing.ErrFileDownloadFailed
	}
	defer reader.Close()

	// Step 3: Parse JSONL → slice of AnalyticsPost
	records, err := uc.parseJSONL(ctx, reader)
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.Index: Failed to parse file: %v", err)
		return indexing.IndexOutput{}, indexing.ErrFileParseFailed
	}

	// Step 4: Process batch (parallel)
	result := uc.processBatch(ctx, input, records)

	// Step 5: Invalidate cache (nếu có records thành công)
	if result.Indexed > 0 {
		if err := uc.invalidateSearchCache(ctx, input.ProjectID); err != nil {
			uc.l.Warnf(ctx, "indexing.usecase.Index: Failed to invalidate cache: %v", err)
		}
	}

	// Step 6: Return output
	result.BatchID = input.BatchID
	result.TotalRecords = len(records)
	result.Duration = time.Since(startTime)

	return result, nil
}

// processBatch - Process batch in parallel with errgroup
func (uc *implUseCase) processBatch(ctx context.Context, input indexing.IndexInput, records []indexing.AnalyticsPost) indexing.IndexOutput {
	var (
		indexed       int
		failed        int
		skipped       int
		mu            sync.Mutex
		failedRecords []indexing.FailedRecord
	)

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(indexing.MaxConcurrency)

	for i := range records {
		record := records[i]

		g.Go(func() error {
			result := uc.indexSingleRecord(gctx, input, record)

			mu.Lock()
			defer mu.Unlock()

			switch result.Status {
			case "indexed":
				indexed++
			case "skipped":
				skipped++
			case "failed":
				failed++
				failedRecords = append(failedRecords, indexing.FailedRecord{
					AnalyticsID:  record.ID,
					ErrorType:    result.ErrorType,
					ErrorMessage: result.ErrorMessage,
				})
			}

			return nil
		})
	}

	_ = g.Wait()

	return indexing.IndexOutput{
		Indexed:       indexed,
		Failed:        failed,
		Skipped:       skipped,
		FailedRecords: failedRecords,
	}
}

// indexSingleRecord - Process single record: validate → dedup → embed → upsert
func (uc *implUseCase) indexSingleRecord(ctx context.Context, ip indexing.IndexInput, record indexing.AnalyticsPost) indexing.IndexRecordResult {
	startTime := time.Now()

	// Step 1: Validate record
	if err := uc.validateAnalyticsPost(record); err != nil {
		return indexing.IndexRecordResult{
			Status:       "skipped",
			ErrorType:    indexing.VALIDATION_ERROR,
			ErrorMessage: err.Error(),
		}
	}

	// Step 2: Pre-filter (spam, bot, quality)
	if uc.shouldSkipRecord(record) {
		return indexing.IndexRecordResult{Status: "skipped"}
	}

	// Step 3: Check duplicate
	contentHash := uc.generateContentHash(record.Content)

	existingDoc, _ := uc.postgreRepo.GetOneDocument(ctx, repo.GetOneDocumentOptions{
		AnalyticsID: record.ID,
	})
	isReindex := existingDoc.ID != ""

	if !isReindex {
		contentDup, _ := uc.postgreRepo.GetOneDocument(ctx, repo.GetOneDocumentOptions{
			ContentHash: contentHash,
		})
		if contentDup.ID != "" {
			return indexing.IndexRecordResult{
				Status:    "skipped",
				ErrorType: indexing.DUPLICATE_CONTENT,
			}
		}
	}

	// Step 4: Create/Update tracking record
	pointID := record.ID
	var trackingDoc model.IndexedDocument
	var err error

	if isReindex {
		trackingDoc, err = uc.postgreRepo.UpsertDocument(ctx, repo.UpsertDocumentOptions{
			AnalyticsID:   record.ID,
			ProjectID:     record.ProjectID,
			SourceID:      record.SourceID,
			QdrantPointID: pointID,
			ContentHash:   contentHash,
			Status:        "PENDING",
			BatchID:       &ip.BatchID,
			RetryCount:    0,
		})
	} else {
		trackingDoc, err = uc.postgreRepo.CreateDocument(ctx, repo.CreateDocumentOptions{
			AnalyticsID:   record.ID,
			ProjectID:     record.ProjectID,
			SourceID:      record.SourceID,
			QdrantPointID: pointID,
			ContentHash:   contentHash,
			Status:        "PENDING",
			BatchID:       &ip.BatchID,
			RetryCount:    0,
		})
	}
	if err != nil {
		return indexing.IndexRecordResult{
			Status:       "failed",
			ErrorType:    indexing.DB_ERROR,
			ErrorMessage: err.Error(),
		}
	}

	// Step 5: Embed content
	embeddingStart := time.Now()
	vector, err := uc.embedContent(ctx, record.Content)
	embeddingTime := int(time.Since(embeddingStart).Milliseconds())

	if err != nil {
		uc.updateFailedStatus(ctx, trackingDoc.ID, indexing.EMBEDDING_ERROR, err.Error(), embeddingTime, 0)
		return indexing.IndexRecordResult{
			Status:       "failed",
			ErrorType:    indexing.EMBEDDING_ERROR,
			ErrorMessage: err.Error(),
		}
	}

	// Step 6: Prepare Qdrant payload
	payload := uc.prepareQdrantPayload(record)

	// Step 7: Upsert to Qdrant
	upsertStart := time.Now()
	err = uc.upsertToQdrant(ctx, pointID, vector, payload)
	upsertTime := int(time.Since(upsertStart).Milliseconds())

	if err != nil {
		uc.updateFailedStatus(ctx, trackingDoc.ID, indexing.QDRANT_ERROR, err.Error(), embeddingTime, upsertTime)
		return indexing.IndexRecordResult{
			Status:       "failed",
			ErrorType:    indexing.QDRANT_ERROR,
			ErrorMessage: err.Error(),
		}
	}

	// Step 8: Update status = INDEXED
	totalTime := int(time.Since(startTime).Milliseconds())
	now := time.Now()
	_, _ = uc.postgreRepo.UpdateDocumentStatus(ctx, repo.UpdateDocumentStatusOptions{
		ID:     trackingDoc.ID,
		Status: indexing.STATUS_INDEXED,
		Metrics: repo.DocumentStatusMetrics{
			IndexedAt:       &now,
			EmbeddingTimeMs: embeddingTime,
			UpsertTimeMs:    upsertTime,
			TotalTimeMs:     totalTime,
		},
	})

	return indexing.IndexRecordResult{Status: indexing.STATUS_INDEXED}
}

// parseJSONL - Parse JSONL file
func (uc *implUseCase) parseJSONL(ctx context.Context, reader io.Reader) ([]indexing.AnalyticsPost, error) {
	var records []indexing.AnalyticsPost
	scanner := bufio.NewScanner(reader)

	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var record indexing.AnalyticsPost
		if err := json.Unmarshal(line, &record); err != nil {
			uc.l.Warnf(ctx, "Failed to parse line %d: %v", lineNum, err)
			continue
		}

		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}

	return records, nil
}

// parseMinIOURL - Parse s3://bucket/path format
func (uc *implUseCase) parseMinIOURL(fileURL string) (bucket, objectName string, err error) {
	if len(fileURL) < 6 || fileURL[:5] != "s3://" {
		return "", "", fmt.Errorf("invalid MinIO URL format: %s", fileURL)
	}

	path := fileURL[5:]
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid MinIO URL format: %s", fileURL)
	}

	return parts[0], parts[1], nil
}

// validateAnalyticsPost - Validate basic fields
func (uc *implUseCase) validateAnalyticsPost(record indexing.AnalyticsPost) error {
	if record.ID == "" {
		return fmt.Errorf("missing analytics_id")
	}
	if record.ProjectID == "" {
		return fmt.Errorf("missing project_id")
	}
	if record.SourceID == "" {
		return fmt.Errorf("missing source_id")
	}
	if len(record.Content) < indexing.MinContentLength {
		return indexing.ErrContentTooShort
	}
	return nil
}

// shouldSkipRecord - Pre-filter: spam, bot, quality
func (uc *implUseCase) shouldSkipRecord(record indexing.AnalyticsPost) bool {
	return record.IsSpam || record.IsBot || record.ContentQualityScore < indexing.MinQualityScore
}

// generateContentHash - Generate SHA-256 hash
func (uc *implUseCase) generateContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// embedContent - Generate embedding with Redis cache
func (uc *implUseCase) embedContent(ctx context.Context, content string) ([]float32, error) {
	cacheKey := uc.getEmbeddingCacheKey(content)

	cachedVector, err := uc.getEmbeddingFromCache(ctx, cacheKey)
	if err == nil && cachedVector != nil {
		uc.l.Debugf(ctx, "indexing.usecase.embedContent: Embedding cache hit")
		return cachedVector, nil
	}

	vectors, err := uc.voyage.Embed(ctx, []string{content})
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.embedContent: Failed to embed content: %v", err)
		return nil, fmt.Errorf("%w: %v", indexing.ErrEmbeddingFailed, err)
	}

	if len(vectors) == 0 {
		return nil, fmt.Errorf("%w: no vectors returned", indexing.ErrEmbeddingFailed)
	}

	vector := vectors[0]

	if err := uc.saveEmbeddingToCache(ctx, cacheKey, vector, 7*24*time.Hour); err != nil {
		uc.l.Warnf(ctx, "indexing.usecase.embedContent: Failed to save embedding to cache: %v", err)
	}

	return vector, nil
}

// getEmbeddingCacheKey - Generate cache key
func (uc *implUseCase) getEmbeddingCacheKey(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("embedding:%x", hash)
}

// getEmbeddingFromCache - Get from Redis
func (uc *implUseCase) getEmbeddingFromCache(ctx context.Context, key string) ([]float32, error) {
	data, err := uc.redis.Get(ctx, key)
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.getEmbeddingFromCache: Failed to get embedding from cache: %v", err)
		return nil, err
	}

	var vector []float32
	if err := json.Unmarshal([]byte(data), &vector); err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.getEmbeddingFromCache: Failed to unmarshal embedding: %v", err)
		return nil, err
	}

	return vector, nil
}

// saveEmbeddingToCache - Save to Redis
func (uc *implUseCase) saveEmbeddingToCache(ctx context.Context, key string, vector []float32, ttl time.Duration) error {
	data, err := json.Marshal(vector)
	if err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.saveEmbeddingToCache: Failed to marshal embedding: %v", err)
		return err
	}
	if err := uc.redis.Set(ctx, key, string(data), ttl); err != nil {
		uc.l.Errorf(ctx, "indexing.usecase.saveEmbeddingToCache: Failed to set embedding in cache: %v", err)
		return err
	}
	return nil
}

// prepareQdrantPayload - Build Qdrant payload from analytics post
func (uc *implUseCase) prepareQdrantPayload(record indexing.AnalyticsPost) map[string]interface{} {
	payload := map[string]interface{}{
		"analytics_id":            record.ID,
		"project_id":              record.ProjectID,
		"source_id":               record.SourceID,
		"content":                 uc.truncateContent(record.Content, 1000),
		"content_created_at":      record.ContentCreatedAt.Unix(),
		"ingested_at":             record.IngestedAt.Unix(),
		"platform":                record.Platform,
		"overall_sentiment":       record.OverallSentiment,
		"overall_sentiment_score": record.OverallSentimentScore,
		"sentiment_confidence":    record.SentimentConfidence,
		"keywords":                record.Keywords,
		"risk_level":              record.RiskLevel,
		"risk_score":              record.RiskScore,
		"requires_attention":      record.RequiresAttention,
		"engagement_score":        record.EngagementScore,
		"virality_score":          record.ViralityScore,
		"influence_score":         record.InfluenceScore,
		"reach_estimate":          record.ReachEstimate,
		"content_quality_score":   record.ContentQualityScore,
		"is_spam":                 record.IsSpam,
		"is_bot":                  record.IsBot,
		"language":                record.Language,
		"toxicity_score":          record.ToxicityScore,
	}

	// Aspects
	if len(record.Aspects) > 0 {
		aspects := make([]map[string]interface{}, len(record.Aspects))
		for i, aspect := range record.Aspects {
			aspects[i] = map[string]interface{}{
				"aspect":              aspect.Aspect,
				"aspect_display_name": aspect.AspectDisplayName,
				"sentiment":           aspect.Sentiment,
				"sentiment_score":     aspect.SentimentScore,
				"keywords":            aspect.Keywords,
				"impact_score":        aspect.ImpactScore,
			}
		}
		payload["aspects"] = aspects
	}

	// Metadata
	metadata := map[string]interface{}{
		"author":              record.UAPMetadata.Author,
		"author_display_name": record.UAPMetadata.AuthorDisplayName,
		"author_followers":    record.UAPMetadata.AuthorFollowers,
		"engagement": map[string]interface{}{
			"views":    record.UAPMetadata.Engagement.Views,
			"likes":    record.UAPMetadata.Engagement.Likes,
			"comments": record.UAPMetadata.Engagement.Comments,
			"shares":   record.UAPMetadata.Engagement.Shares,
		},
	}
	if record.UAPMetadata.VideoURL != "" {
		metadata["video_url"] = record.UAPMetadata.VideoURL
	}
	if len(record.UAPMetadata.Hashtags) > 0 {
		metadata["hashtags"] = record.UAPMetadata.Hashtags
	}
	if record.UAPMetadata.Location != "" {
		metadata["location"] = record.UAPMetadata.Location
	}
	payload["metadata"] = metadata

	return payload
}

// upsertToQdrant - Upsert point via vector repo (collection name nằm ở tầng repo)
func (uc *implUseCase) upsertToQdrant(
	ctx context.Context,
	pointID string,
	vector []float32,
	payload map[string]interface{},
) error {
	if err := uc.vectorRepo.UpsertPoint(ctx, repo.UpsertPointOptions{
		PointID: pointID,
		Vector:  vector,
		Payload: payload,
	}); err != nil {
		return fmt.Errorf("%w: %v", indexing.ErrQdrantUpsertFailed, err)
	}
	return nil
}

// truncateContent - Truncate content to max length
func (uc *implUseCase) truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}

// updateFailedStatus - Update document status to FAILED
func (uc *implUseCase) updateFailedStatus(
	ctx context.Context,
	docID string,
	errorType, errorMessage string,
	embeddingTime, upsertTime int,
) {
	_, _ = uc.postgreRepo.UpdateDocumentStatus(ctx, repo.UpdateDocumentStatusOptions{
		ID:     docID,
		Status: "FAILED",
		Metrics: repo.DocumentStatusMetrics{
			ErrorMessage:    fmt.Sprintf("[%s] %s", errorType, errorMessage),
			RetryCount:      0,
			EmbeddingTimeMs: embeddingTime,
			UpsertTimeMs:    upsertTime,
		},
	})
}

// invalidateSearchCache - Invalidate search cache for project
func (uc *implUseCase) invalidateSearchCache(ctx context.Context, projectID string) error {
	uc.l.Debugf(ctx, "Cache invalidation skipped (DeleteByPattern not implemented)")
	return nil
}
