//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"
	"vnp-memory/services/cognee-search/adapter/client"
	"vnp-memory/services/cognee-search/adapter/grpc"
	"vnp-memory/services/cognee-search/adapter/nats"
	"vnp-memory/services/cognee-search/adapter/repository/neo4j"
	"vnp-memory/services/cognee-search/adapter/repository/qdrant"
	"vnp-memory/services/cognee-search/adapter/repository/redis"
	"vnp-memory/services/cognee-search/adapter/retriever"
	"vnp-memory/services/cognee-search/internal/infrastructure/config"
	infra_grpc "vnp-memory/services/cognee-search/internal/infrastructure/grpc"
	"vnp-memory/services/cognee-search/internal/usecase"
	"vnp-memory/services/cognee-search/internal/usecase/port"
)

var ConfigSet = wire.NewSet(
	config.LoadConfig,
)

var AdapterSet = wire.NewSet(
	// Repositories
	qdrant.NewVectorSearcher,
	neo4j.NewGraphSearcher,
	
	// Redis needs config TTL
	wire.Struct(new(redis.RedisConfigProvider), "*"),
	
	// Clients
	client.NewLLMClient,
	client.NewRerankerClient,
	
	// Retrievers
	retriever.NewSimilarityRetriever,
	retriever.NewChunksRetriever,
	retriever.NewCypherRetriever,
	
	// Registry (we construct the slice of retrievers first)
	ProvideRetrievers,
	retriever.NewRegistry,
	wire.Bind(new(usecase.Registry), new(*retriever.Registry)),
	
	// Handlers
	grpc.NewHandler,
	nats.NewSubscriber,
)

var UsecaseSet = wire.NewSet(
	usecase.NewSearchUseCase,
	usecase.NewRAGCompleteUseCase,
	usecase.NewExploreGraphUseCase,
)

var ServerSet = wire.NewSet(
	infra_grpc.NewServer,
)

func ProvideRetrievers(
	sim port.Retriever, 
	chunks port.Retriever, 
	cypher port.Retriever,
) []port.Retriever {
	return []port.Retriever{sim, chunks, cypher}
}

func InitializeServer() (*infra_grpc.Server, *nats.Subscriber, error) {
	wire.Build(
		ConfigSet,
		AdapterSet,
		UsecaseSet,
		ServerSet,
	)
	return nil, nil, nil
}
