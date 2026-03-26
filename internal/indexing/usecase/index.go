package usecase

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"knowledge-srv/internal/embedding"
	"knowledge-srv/internal/indexing"
	repo "knowledge-srv/internal/indexing/repository"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/point"
	"strings"
	"sync"
	"time"

	"github.com/smap-hcmut/shared-libs/go/minio"
	"golang.org/x/sync/errgroup"
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
	// Keeps using CacheRepository (Redis) because InvalidateSearchCache is specific to existing Redis repo logic
	// Ideally this should move to Search or Point domain too, but kept here for scope minimization.
	// Actually, point domain doesn't expose Invalidate.
	if result.Indexed > 0 {
		if err := uc.cacheRepo.InvalidateSearchCache(ctx, input.ProjectID); err != nil {
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

	// Step 5: Embed content (Via Embedding Domain)
	embeddingStart := time.Now()
	// Call Embedding Domain directly
	generateOutput, err := uc.embeddingUC.Generate(ctx, embedding.GenerateInput{
		Text: record.Content,
	})
	vector := generateOutput.Vector
	embeddingTime := int(time.Since(embeddingStart).Milliseconds())

	if err != nil {
		uc.updateFailedStatus(ctx, trackingDoc.ID, indexing.EMBEDDING_ERROR, err.Error(), embeddingTime, 0)
		uc.writeToDLQ(ctx, record, ip.BatchID, indexing.EMBEDDING_ERROR, err.Error(), trackingDoc.RetryCount)
		return indexing.IndexRecordResult{
			Status:       "failed",
			ErrorType:    indexing.EMBEDDING_ERROR,
			ErrorMessage: err.Error(),
		}
	}

	// Step 6: Prepare Qdrant payload
	payload := uc.prepareQdrantPayload(record)

	// Step 7: Upsert to Qdrant (Via Point Domain)
	upsertStart := time.Now()
	// Call Point Domain
	err = uc.pointUC.Upsert(ctx, point.UpsertInput{
		CollectionName: fmt.Sprintf("proj_%s", ip.ProjectID),
		Points: []model.Point{
			{
				ID:      pointID,
				Vector:  vector,
				Payload: payload,
			},
		},
	})
	upsertTime := int(time.Since(upsertStart).Milliseconds())

	if err != nil {
		uc.updateFailedStatus(ctx, trackingDoc.ID, indexing.QDRANT_ERROR, err.Error(), embeddingTime, upsertTime)
		uc.writeToDLQ(ctx, record, ip.BatchID, indexing.QDRANT_ERROR, err.Error(), trackingDoc.RetryCount)
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
			uc.l.Warnf(ctx, "indexing.usecase.Index: Failed to parse line %d: %v", lineNum, err)
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

// prepareQdrantPayload - Build Qdrant payload from analytics post
func (uc *implUseCase) prepareQdrantPayload(record indexing.AnalyticsPost) map[string]interface{} {
	payload := analyticsPayload{
		AnalyticsID:           record.ID,
		ProjectID:             record.ProjectID,
		SourceID:              record.SourceID,
		Content:               uc.truncateContent(record.Content, 1000),
		ContentCreatedAt:      record.ContentCreatedAt.Unix(),
		IngestedAt:            record.IngestedAt.Unix(),
		Platform:              record.Platform,
		OverallSentiment:      record.OverallSentiment,
		OverallSentimentScore: record.OverallSentimentScore,
		SentimentConfidence:   record.SentimentConfidence,
		Keywords:              record.Keywords,
		RiskLevel:             record.RiskLevel,
		RiskScore:             record.RiskScore,
		RequiresAttention:     record.RequiresAttention,
		EngagementScore:       record.EngagementScore,
		ViralityScore:         record.ViralityScore,
		InfluenceScore:        record.InfluenceScore,
		ReachEstimate:         record.ReachEstimate,
		ContentQualityScore:   record.ContentQualityScore,
		IsSpam:                record.IsSpam,
		IsBot:                 record.IsBot,
		Language:              record.Language,
		ToxicityScore:         record.ToxicityScore,
		Aspects:               mapAnalyticsAspects(record.Aspects),
		Metadata: analyticsMetadataPayload{
			Author:            record.UAPMetadata.Author,
			AuthorDisplayName: record.UAPMetadata.AuthorDisplayName,
			AuthorFollowers:   record.UAPMetadata.AuthorFollowers,
			Engagement: analyticsEngagementPayload{
				Views:    record.UAPMetadata.Engagement.Views,
				Likes:    record.UAPMetadata.Engagement.Likes,
				Comments: record.UAPMetadata.Engagement.Comments,
				Shares:   record.UAPMetadata.Engagement.Shares,
			},
			VideoURL: record.UAPMetadata.VideoURL,
			Hashtags: record.UAPMetadata.Hashtags,
			Location: record.UAPMetadata.Location,
		},
	}

	return uc.payloadFromStruct(payload)
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

// writeToDLQ - Write failed record to Dead Letter Queue
func (uc *implUseCase) writeToDLQ(
	ctx context.Context,
	record indexing.AnalyticsPost,
	batchID string,
	errorType, errorMessage string,
	retryCount int,
) {
	_, err := uc.postgreRepo.CreateDLQ(ctx, repo.CreateDLQOptions{
		AnalyticsID:  record.ID,
		ProjectID:    record.ProjectID,
		SourceID:     record.SourceID,
		ContentHash:  uc.generateContentHash(record.Content),
		ErrorType:    errorType,
		ErrorMessage: errorMessage,
		RetryCount:   retryCount,
		BatchID:      &batchID,
		FailedAt:     time.Now(),
	})
	if err != nil {
		uc.l.Warnf(ctx, "indexing.usecase.Index: writeToDLQ failed: %v", err)
	}
}
