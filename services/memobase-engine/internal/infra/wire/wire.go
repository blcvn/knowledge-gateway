//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/google/wire"
	"vnp-memory/services/memobase-engine/internal/adapter/client"
	"vnp-memory/services/memobase-engine/internal/adapter/event"
	adapter "vnp-memory/services/memobase-engine/internal/adapter/grpc"
	"vnp-memory/services/memobase-engine/internal/adapter/repository/postgres"
	"vnp-memory/services/memobase-engine/internal/infra/config"
	"vnp-memory/services/memobase-engine/internal/infra/server"
	"vnp-memory/services/memobase-engine/internal/usecase"
)

// InitializeApp wires all dependencies to build the gRPC server.
func InitializeApp(cfg *config.Config) (*server.GRPCServer, error) {
	wire.Build(
		// Repositories
		postgres.NewProfileRepository,
		postgres.NewEventRepository,
		postgres.NewEventGistRepository,
		// blob repository needs an implementation

		// Clients & Event
		client.NewBifrostClient,
		client.NewEmbedderClient,
		event.NewNatsPublisher,

		// Use Cases
		usecase.NewExtractProfileUseCase,
		usecase.NewMergeProfileUseCase,
		usecase.NewProcessEventUseCase,
		usecase.NewProcessBufferUseCase,

		// Adapters & Server
		adapter.NewEngineHandler,
		server.NewGRPCServer,
	)
	return &server.GRPCServer{}, nil
}
