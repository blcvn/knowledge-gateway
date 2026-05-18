package handler

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/vnp-community/vnp-memory/gateway/internal/domain"
)

// noopTestRegistry is a mock ServiceRegistry for unit tests.
type noopTestRegistry struct{}

func (r *noopTestRegistry) Resolve(service string) (*domain.RouteTarget, error) {
	return &domain.RouteTarget{
		Service: service,
		Address: "localhost:0",
		Timeout: 5 * time.Second,
	}, nil
}

func (r *noopTestRegistry) Forward(_ context.Context, _ *domain.RouteTarget, _ []byte) ([]byte, error) {
	return []byte(`{"status":"ok"}`), nil
}

func (r *noopTestRegistry) HealthCheck(_ string) (bool, error) {
	return true, nil
}

// testLogger returns a discarding logger for unit tests.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}
