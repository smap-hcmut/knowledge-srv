package consumer

import (
	"context"
)

// domainConsumers holds references to all domain consumers for cleanup
type domainConsumers struct {
	// Add domain consumers here as needed
	// Example: searchConsumer *searchConsumer.Consumer
}

// setupDomains initializes all domain layers (repositories, usecases, consumers)
func (srv *ConsumerServer) setupDomains(ctx context.Context) (*domainConsumers, error) {
	// Initialize domains here as needed
	// Example:
	// searchRepo := searchRepo.New(srv.postgresDB)
	// searchUC := searchUsecase.New(srv.l, searchRepo, ...)
	// searchCons, err := searchConsumer.New(...)

	return &domainConsumers{
		// Add initialized consumers here
	}, nil
}

// startConsumers starts all domain consumers in background goroutines
func (srv *ConsumerServer) startConsumers(ctx context.Context, consumers *domainConsumers) error {
	// Start consumers here as needed
	// Example:
	// if err := consumers.searchConsumer.ConsumeSearchRequests(ctx); err != nil {
	//     return fmt.Errorf("failed to start search consumer: %w", err)
	// }

	srv.l.Infof(ctx, "All consumers started successfully")
	return nil
}

// stopConsumers gracefully stops all domain consumers
func (srv *ConsumerServer) stopConsumers(ctx context.Context, consumers *domainConsumers) {
	// Close consumers here as needed
	// Example:
	// if consumers.searchConsumer != nil {
	//     if err := consumers.searchConsumer.Close(); err != nil {
	//         srv.l.Errorf(ctx, "Error closing search consumer: %v", err)
	//     }
	// }

	srv.l.Infof(ctx, "All consumers stopped")
}
