//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"

	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/adapter/event"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/adapter/repository/postgres"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/infra/llm"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/infra/server"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/usecase/engine"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/usecase/ingestion"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/usecase/port"
)

var RepoSet = wire.NewSet(
	postgres.NewBlobRepo,
	wire.Bind(new(port.BlobRepository), new(*postgres.BlobRepo)),
	postgres.NewBufferRepo,
	wire.Bind(new(port.BufferRepository), new(*postgres.BufferRepo)),
	postgres.NewProfileRepo,
	wire.Bind(new(port.ProfileRepository), new(*postgres.ProfileRepo)),
	postgres.NewGistRepo,
	wire.Bind(new(port.GistRepository), new(*postgres.GistRepo)),
)

var InfraSet = wire.NewSet(
	llm.NewBifrostClient,
	wire.Bind(new(port.LLMService), new(*llm.BifrostClient)),
	event.NewNATSPublisher,
	wire.Bind(new(port.EventPublisher), new(*event.NATSPublisher)),
)

var UsecaseSet = wire.NewSet(
	engine.NewService,
	wire.Bind(new(port.EngineUseCase), new(*engine.Service)),
	ingestion.NewService,
	wire.Bind(new(port.IngestionUseCase), new(*ingestion.Service)),
)

// InitializeServer initializes the gRPC Server and its dependencies.
func InitializeServer(db *pgxpool.Pool, nc *nats.Conn, bifrostEndpoint string) (*server.IngestionHandler, error) {
	wire.Build(
		RepoSet,
		InfraSet,
		UsecaseSet,
		server.NewIngestionHandler,
	)
	return &server.IngestionHandler{}, nil
}
