package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/vnp-community/vnp-memory/services/cognee-pipeline/internal/infra/server"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func main() {
	// Initialize enterprise logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Bootstrapping cognee-pipeline service...")

	// Dependency Injection (Simulating wire.go initialization)
	// TODO: Replace with wire InitializeServer
	srv := server.NewGRPCServer(nil, logger) // passing nil handler for brevity in this scaffold

	// Setup Graceful Shutdown context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, gCtx := errgroup.WithContext(ctx)

	// Start server
	g.Go(func() error {
		return srv.Start(gCtx)
	})

	// Listen for termination signals
	g.Go(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		select {
		case sig := <-c:
			logger.Info("Received termination signal", zap.String("signal", sig.String()))
			cancel()
			srv.Stop()
		case <-gCtx.Done():
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error("Service exited with error", zap.Error(err))
	} else {
		logger.Info("Service gracefully stopped")
	}
}
