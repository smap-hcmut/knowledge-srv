package consumer

import (
	"context"
	"fmt"

	indexingConsumer "knowledge-srv/internal/indexing/delivery/kafka/consumer"
	indexingRepo "knowledge-srv/internal/indexing/repository/postgre"
	indexingUsecase "knowledge-srv/internal/indexing/usecase"
)

// domainConsumers holds references to all domain consumers for cleanup
type domainConsumers struct {
	indexingConsumer *indexingConsumer.Consumer
}

// setupDomains initializes all domain layers (repositories, usecases, consumers)
func (srv *ConsumerServer) setupDomains(ctx context.Context) (*domainConsumers, error) {
	// Initialize indexing domain
	indexingRepo := indexingRepo.New(srv.postgresDB)
	indexingUC := indexingUsecase.New(
		srv.l,
		indexingRepo,
		srv.qdrantClient,
		srv.minioClient,
		srv.voyageClient,
		srv.redisClient,
		"knowledge_indexing", // TODO: move to config
	)
	indexingCons, err := indexingConsumer.New(indexingConsumer.Config{
		Logger:      srv.l,
		KafkaConfig: srv.kafkaConfig,
		UseCase:     indexingUC,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create indexing consumer: %w", err)
	}

	srv.l.Infof(ctx, "Indexing domain initialized")

	return &domainConsumers{
		indexingConsumer: indexingCons,
	}, nil
}

// startConsumers starts all domain consumers in background goroutines
func (srv *ConsumerServer) startConsumers(ctx context.Context, consumers *domainConsumers) error {
	// Start indexing consumer
	topic := "analytics.batch.completed" // TODO: move to config
	if err := consumers.indexingConsumer.ConsumeBatchCompleted(ctx, topic); err != nil {
		return fmt.Errorf("failed to start indexing consumer: %w", err)
	}

	srv.l.Infof(ctx, "All consumers started successfully")
	return nil
}

// stopConsumers gracefully stops all domain consumers
func (srv *ConsumerServer) stopConsumers(ctx context.Context, consumers *domainConsumers) {
	// Close indexing consumer
	if consumers.indexingConsumer != nil {
		if err := consumers.indexingConsumer.Close(); err != nil {
			srv.l.Errorf(ctx, "Error closing indexing consumer: %v", err)
		}
	}

	srv.l.Infof(ctx, "All consumers stopped")
}
