package consumer

import (
	"context"
	"fmt"

	indexingConsumer "knowledge-srv/internal/indexing/delivery/kafka/consumer"
	indexingProducer "knowledge-srv/internal/indexing/delivery/kafka/producer"
	indexingRepo "knowledge-srv/internal/indexing/repository/postgre"
	indexingUsecase "knowledge-srv/internal/indexing/usecase"
)

// domainConsumers holds references to all domain consumers for cleanup
type domainConsumers struct {
	indexingConsumer *indexingConsumer.Consumer
	// Add more domain consumers here as needed
}

// setupDomains initializes all domain layers (repositories, usecases, consumers)
func (srv *ConsumerServer) setupDomains(ctx context.Context) (*domainConsumers, error) {
	// Initialize Indexing domain
	indexingProd := indexingProducer.New(srv.l, srv.kafkaProducer)
	indexingRepository := indexingRepo.New(srv.postgresDB)
	indexingUC := indexingUsecase.New(
		srv.l,
		indexingRepository,
		srv.qdrantClient,
		srv.minioClient,
		srv.voyageClient,
		srv.geminiClient,
		srv.redisClient,
		indexingProd,
		srv.discord,
	)

	indexingCons, err := indexingConsumer.New(indexingConsumer.Config{
		Logger:      srv.l,
		KafkaConfig: srv.kafkaConfig,
		UseCase:     indexingUC,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create indexing consumer: %w", err)
	}

	// Add more domains here...

	return &domainConsumers{
		indexingConsumer: indexingCons,
	}, nil
}

// startConsumers starts all domain consumers in background goroutines
func (srv *ConsumerServer) startConsumers(ctx context.Context, consumers *domainConsumers) error {
	// Start document indexing consumer
	if err := consumers.indexingConsumer.ConsumeDocumentIndexing(ctx); err != nil {
		return fmt.Errorf("failed to start document indexing consumer: %w", err)
	}

	// Start knowledge base events consumer
	if err := consumers.indexingConsumer.ConsumeKnowledgeBaseEvents(ctx); err != nil {
		return fmt.Errorf("failed to start knowledge base events consumer: %w", err)
	}

	// Add more consumers here...

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

	// Add more consumer cleanup here...
}
