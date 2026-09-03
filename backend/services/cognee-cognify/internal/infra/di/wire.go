//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"
	"github.com/nats-io/nats.go"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"gorm.io/gorm"

	"vnp-memory/services/cognee-cognify/internal/adapter/client"
	grpc_adapter "vnp-memory/services/cognee-cognify/internal/adapter/grpc"
	nats_adapter "vnp-memory/services/cognee-cognify/internal/adapter/nats"
	"vnp-memory/services/cognee-cognify/internal/adapter/repository/neo4j_repo"
	"vnp-memory/services/cognee-cognify/internal/adapter/repository/postgres"
	"vnp-memory/services/cognee-cognify/internal/adapter/repository/qdrant"
	"vnp-memory/services/cognee-cognify/internal/infra/config"
	"vnp-memory/services/cognee-cognify/internal/usecase"
	"vnp-memory/services/cognee-cognify/internal/usecase/port"
)

func InitializeApp(cfg *config.Config, db *gorm.DB, neo4jDriver neo4j.DriverWithContext, nc *nats.Conn) (grpc_adapter.CogneeCognifyServiceServer, *nats_adapter.DataIngestedSubscriber, error) {
	wire.Build(
		postgres.NewJobRepository,
		// Provide Qdrant
		wire.Bind(new(port.VectorRepository), new(*qdrant.VectorRepository)), // Placeholder
		
		// Setup Neo4j
		neo4j_repo.NewGraphRepository,
		
		// Providers
		client.NewLLMClient,
		client.NewEmbedderClient,
		nats_adapter.NewEventPublisher,
		
		usecase.NewCognifyOrchestrator,
		grpc_adapter.NewHandler,
		nats_adapter.NewDataIngestedSubscriber,
	)
	return nil, nil, nil
}
