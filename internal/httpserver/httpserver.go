package httpserver

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// Run starts the HTTP server and blocks until a shutdown signal is received.
func (srv HTTPServer) Run() error {
	err := srv.mapHandlers()
	if err != nil {
		srv.l.Fatalf(context.Background(), "Failed to map handlers: %v", err)
		return err
	}

	ctx := context.Background()
	go func() {
		srv.gin.Run(fmt.Sprintf("%s:%d", srv.host, srv.port))
	}()

	srv.l.Infof(ctx, "Started server on : %s:%d", srv.host, srv.port)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	srv.l.Info(ctx, <-ch)
	srv.l.Info(ctx, "Stopping API server.")

	return nil
}
