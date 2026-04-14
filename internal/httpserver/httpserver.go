package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Run starts the HTTP server and blocks until the context is cancelled.
// On context cancellation, it performs graceful shutdown with a 15s deadline.
func (srv HTTPServer) Run(ctx context.Context) error {
	if err := srv.mapHandlers(); err != nil {
		return fmt.Errorf("map handlers: %w", err)
	}

	addr := fmt.Sprintf(":%d", srv.port)
	server := &http.Server{
		Addr:    addr,
		Handler: srv.gin,
	}

	// Graceful shutdown goroutine — triggers when parent context is cancelled
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			srv.l.Errorf(ctx, "Server shutdown error: %v", err)
		}
	}()

	srv.l.Infof(ctx, "HTTP server listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	srv.l.Info(ctx, "HTTP server stopped")
	return nil
}
