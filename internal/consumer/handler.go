package consumer

import (
	"context"
	"fmt"

	embeddingRepo "knowledge-srv/internal/embedding/repository/redis"
	embeddingUsecase "knowledge-srv/internal/embedding/usecase"
	indexingConsumer "knowledge-srv/internal/indexing/delivery/kafka/consumer"
	indexingPostgre "knowledge-srv/internal/indexing/repository/postgre"
	indexingRedis "knowledge-srv/internal/indexing/repository/redis"
	indexingUsecase "knowledge-srv/internal/indexing/usecase"
	"knowledge-srv/internal/model"
	"knowledge-srv/internal/notebook"
	notebookPostgre "knowledge-srv/internal/notebook/repository/postgre"
	notebookUsecase "knowledge-srv/internal/notebook/usecase"
	pointRepo "knowledge-srv/internal/point/repository/qdrant"
	pointUsecase "knowledge-srv/internal/point/usecase"
	"knowledge-srv/internal/transform"
	transformUsecase "knowledge-srv/internal/transform/usecase"
)

// domainConsumers holds references to all domain consumers for cleanup (interface, like http.Handler)
type domainConsumers struct {
	indexingConsumer indexingConsumer.Consumer
	notebookUC       notebook.UseCase
}

// setupDomains initializes all domain layers (repositories, usecases, consumers)
func (srv *ConsumerServer) setupDomains(ctx context.Context) (*domainConsumers, error) {
	// 1. Core Domains (Embedding, Point)
	// Embedding
	embeddingCacheRepo := embeddingRepo.New(srv.redisClient, srv.l)
	embeddingUC := embeddingUsecase.New(
		embeddingCacheRepo,
		srv.voyageClient,
		srv.l,
	)

	// Point
	pointQdrantRepo := pointRepo.New(srv.qdrantClient, srv.l)
	pointUC := pointUsecase.New(
		pointQdrantRepo,
		srv.l,
	)

	// 2. Indexing Domain
	postgreRepo := indexingPostgre.New(srv.postgresDB, srv.l)
	cacheRepo := indexingRedis.New(srv.redisClient, srv.l)

	indexingUC := indexingUsecase.New(
		srv.l,
		postgreRepo,
		pointUC,
		embeddingUC,
		cacheRepo,
		srv.minioClient,
	)

	var notebookUC notebook.UseCase
	var transformUC transform.UseCase
	if srv.appConfig != nil && srv.appConfig.Notebook.Enabled && srv.maestroClient != nil {
		transformUC = transformUsecase.New(pointUC, srv.appConfig.Notebook.MaxPostsPerPart, srv.l)
		campaignRepo := notebookPostgre.NewCampaignRepo(srv.postgresDB)
		sourceRepo := notebookPostgre.NewSourceRepo(srv.postgresDB)
		sessionRepo := notebookPostgre.NewSessionRepo()
		chatJobRepo := notebookPostgre.NewChatJobRepo(srv.postgresDB)
		notebookUC = notebookUsecase.NewUseCase(
			srv.maestroClient,
			campaignRepo,
			sourceRepo,
			sessionRepo,
			chatJobRepo,
			notebookUsecase.Config{
				NotebookEnabled:    srv.appConfig.Notebook.Enabled,
				JobPollIntervalMs:  srv.appConfig.Maestro.JobPollIntervalMs,
				JobPollMaxAttempts: srv.appConfig.Maestro.JobPollMaxAttempts,
				SyncMaxRetries:     srv.appConfig.Notebook.SyncMaxRetries,
				WebhookCallbackURL: srv.appConfig.Maestro.WebhookCallbackURL,
				WebhookSecret:      srv.appConfig.Maestro.WebhookSecret,
			},
			srv.l,
		)
		if err := notebookUC.StartSessionLoop(ctx, model.Scope{}); err != nil {
			srv.l.Warnf(ctx, "consumer.handler.setupDomains: notebook session start failed (non-fatal): %v", err)
			notebookUC = nil
			transformUC = nil
		}
	}

	indexingCons, err := indexingConsumer.New(indexingConsumer.Config{
		Logger:      srv.l,
		KafkaConfig: srv.kafkaConfig,
		UseCase:     indexingUC,
		NotebookUC:  notebookUC,
		TransformUC: transformUC,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create indexing consumer: %w", err)
	}

	srv.l.Infof(ctx, "Indexing domain initialized")

	return &domainConsumers{
		indexingConsumer: indexingCons,
		notebookUC:       notebookUC,
	}, nil
}

// startConsumers starts all domain consumers in background goroutines
func (srv *ConsumerServer) startConsumers(ctx context.Context, consumers *domainConsumers) error {
	// Start indexing consumer
	if err := consumers.indexingConsumer.ConsumeBatchCompleted(ctx); err != nil {
		return fmt.Errorf("failed to start indexing consumer: %w", err)
	}

	if err := consumers.indexingConsumer.ConsumeInsightsPublished(ctx); err != nil {
		return fmt.Errorf("failed to start insights consumer: %w", err)
	}

	if err := consumers.indexingConsumer.ConsumeReportDigest(ctx); err != nil {
		return fmt.Errorf("failed to start digest consumer: %w", err)
	}

	srv.l.Infof(ctx, "All consumers started successfully")
	return nil
}

// stopConsumers gracefully stops all domain consumers
func (srv *ConsumerServer) stopConsumers(ctx context.Context, consumers *domainConsumers) {
	if consumers.notebookUC != nil {
		if err := consumers.notebookUC.StopSessionLoop(ctx, model.Scope{}); err != nil {
			srv.l.Warnf(ctx, "consumer.stopConsumers: notebook StopSessionLoop: %v", err)
		}
	}
	// Close indexing consumer
	if consumers.indexingConsumer != nil {
		if err := consumers.indexingConsumer.Close(); err != nil {
			srv.l.Errorf(ctx, "Error closing indexing consumer: %v", err)
		}
	}

	srv.l.Infof(ctx, "All consumers stopped")
}
