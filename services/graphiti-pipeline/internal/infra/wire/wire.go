//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/google/wire"

	"graphiti-pipeline/internal/adapter/client"
	"graphiti-pipeline/internal/adapter/embedder"
	"graphiti-pipeline/internal/adapter/event"
	grpc_adapter "graphiti-pipeline/internal/adapter/grpc"
	"graphiti-pipeline/internal/adapter/llm"
	"graphiti-pipeline/internal/adapter/repository/neo4j"
	"graphiti-pipeline/internal/adapter/repository/postgres"
	"graphiti-pipeline/internal/infra/config"
	"graphiti-pipeline/internal/infra/server"
	"graphiti-pipeline/internal/infra/telemetry"
	"graphiti-pipeline/internal/usecase/ingest"
	"graphiti-pipeline/internal/usecase/knowledge"
	"graphiti-pipeline/internal/usecase/saga"
	"google.golang.org/grpc"
)

var InfraSet = wire.NewSet(
	config.LoadConfig,
	server.NewGRPCServer,
	telemetry.NewLogger,
	telemetry.NewTracer,
	telemetry.NewMetrics,
)

var AdapterSet = wire.NewSet(
	grpc_adapter.NewIngestionHandler,
	grpc_adapter.NewKnowledgeHandler,
	llm.NewBifrostClient,
	embedder.NewBifrostEmbedder,
	postgres.NewEpisodeRepo,
	postgres.NewSagaRepo,
	postgres.NewGroupLock,
	neo4j.NewEntityReader,
	event.NewNATSPublisher,
	client.NewStoreClient,
)

var UsecaseSet = wire.NewSet(
	ingest.NewIngestEpisodeUseCase,
	ingest.NewBulkIngestUseCase,
	knowledge.NewKnowledgeUsecase,
	saga.NewSagaOrchestrator,
)

type App struct {
	Server *grpc.Server
}

func InitializeApp() (App, error) {
	panic(wire.Build(InfraSet, AdapterSet, UsecaseSet, wire.Struct(new(App), "Server")))
}
