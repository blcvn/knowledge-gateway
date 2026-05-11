//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/google/wire"
	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/adapter/client"
	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/adapter/event"
	adaptergrpc "github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/adapter/grpc"
	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/adapter/repository/postgres"
	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/usecase"
	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/infra/config"
	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/infra/server"
)

func InitializeApp() (*server.GRPCServer, error) {
	wire.Build(
		config.LoadConfig,
		// Mock DB/NATS connection providers here
		postgres.NewSagaRepo,
		wire.Bind(new(usecase.SagaStateRepo), new(*postgres.SagaRepo)),
		postgres.NewEpisodeRepo,
		wire.Bind(new(usecase.EpisodeRepo), new(*postgres.EpisodeRepo)),
		
		event.NewNatsPublisher,
		wire.Bind(new(usecase.EventPublisher), new(*event.NatsPublisher)),

		client.NewKnowledgeClientAdapter,
		wire.Bind(new(usecase.KnowledgeClient), new(*client.KnowledgeClientAdapter)),
		
		client.NewStoreClientAdapter,
		wire.Bind(new(usecase.StoreClient), new(*client.StoreClientAdapter)),

		usecase.NewSagaOrchestrator,
		usecase.NewIngestEpisodeUseCase,
		usecase.NewGetStatusUseCase,
		usecase.NewBulkIngestUseCase,
		
		adaptergrpc.NewHandler,
		
		// Map config to server instantiation
		wire.Struct(new(server.GRPCServer), "*"), // mock
	)
	return nil, nil
}
