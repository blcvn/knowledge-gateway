//go:build wireinject
// +build wireinject

// Package wire defines the Google Wire dependency injection providers
// for the cognee-ingestion service.
//
// Run `wire ./internal/infra/wire/` to generate wire_gen.go.
package wire

import (
	"context"
	"log/slog"

	"vnp-memory/services/cognee-ingestion/internal/adapter/extractor"
	"vnp-memory/services/cognee-ingestion/internal/adapter/hash"
	"vnp-memory/services/cognee-ingestion/internal/infra/config"
	"vnp-memory/services/cognee-ingestion/internal/infra/server"
	"vnp-memory/services/cognee-ingestion/internal/infra/telemetry"
)

// App holds all top-level application components.
type App struct {
	Server *server.Server
	Config *config.Config
	Logger *slog.Logger
}

// InitializeApp wires up the entire application.
// This is a Wire injector — the implementation is generated.
func InitializeApp(ctx context.Context) (*App, error) {
	panic("wire: this function body should be replaced by wire_gen.go")

	// Wire provider hints (not executed, just for Wire analysis):
	// wire.Build(
	//     config.Load,
	//     ProvideTelemetry,
	//     ProvideServer,
	//     ProvideRepositories,
	//     ProvideAdapters,
	//     ProvideUseCases,
	//     ProvideHandler,
	//     wire.Struct(new(App), "*"),
	// )
}

// ProvideTelemetry creates the logger from config.
func ProvideTelemetry(cfg *config.Config) *slog.Logger {
	return telemetry.NewLogger(cfg.Telemetry.LogLevel, cfg.Telemetry.LogFormat)
}

// ProvideServer creates the gRPC + health server.
func ProvideServer(cfg *config.Config, logger *slog.Logger) *server.Server {
	return server.New(cfg, logger)
}

// ProvideExtractorRegistry creates the text extractor registry.
func ProvideExtractorRegistry() *extractor.Registry {
	return extractor.NewRegistry()
}

// ProvideHashComputer creates the SHA-256 hash computer.
func ProvideHashComputer() *hash.SHA256Computer {
	return hash.NewSHA256Computer()
}

// NOTE: The remaining providers (PostgreSQL pool, MinIO client, NATS conn,
// repositories, usecases, gRPC handler) require runtime connections.
// They follow the same pattern:
//
//   func ProvidePostgresPool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
//       return pgxpool.New(ctx, cfg.Postgres.URL)
//   }
//
//   func ProvideMinIOClient(cfg *config.Config) (*minio.Client, error) {
//       return minio.New(cfg.MinIO.Endpoint, &minio.Options{...})
//   }
//
//   func ProvideNATSConn(cfg *config.Config) (*nats.Conn, error) {
//       return nats.Connect(cfg.NATS.URL)
//   }
//
// Once these are wired, run: go run github.com/google/wire/cmd/wire ./internal/infra/wire/
