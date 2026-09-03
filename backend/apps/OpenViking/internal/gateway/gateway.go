package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/vnp-community/vnp-memory/apps/OpenViking/internal/config"
)

func Run(ctx context.Context, logger *slog.Logger) error {
	cfg := config.Load()
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"alive"}`))
	})

	// In a complete implementation, this gateway proxy will forward HTTP REST
	// requests to the underlying gRPC localhost services based on the routing map.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(`{"error": "route pending grpc transcoding"}`))
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.RESTPort),
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("embedded gateway proxy listening", "port", cfg.Server.RESTPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
