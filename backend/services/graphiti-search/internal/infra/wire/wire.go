//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/google/wire"

	"vnp-memory/services/graphiti-search/internal/adapter/cache"
	"vnp-memory/services/graphiti-search/internal/adapter/client"
	"vnp-memory/services/graphiti-search/internal/adapter/event"
	searchgrpc "vnp-memory/services/graphiti-search/internal/adapter/grpc"
	"vnp-memory/services/graphiti-search/internal/infra/config"
	"vnp-memory/services/graphiti-search/internal/infra/server"
	"vnp-memory/services/graphiti-search/internal/usecase"
	"vnp-memory/services/graphiti-search/internal/usecase/reranker"
)

func InitializeServer(cfg *config.Config) (*server.GRPCServer, *event.NatsSubscriber, error) {
	wire.Build(
		client.NewStoreClientAdapter,
		cache.NewRedisCacheAdapter,
		
		reranker.NewRRFReranker,
		reranker.NewMMRReranker,
		reranker.NewCrossEncoderReranker,
		reranker.NewNodeDistanceReranker,
		reranker.NewEpisodeMentionsReranker,

		wire.Struct(new(usecase.EmbedderClient), "*"),
		
		usecase.NewHybridSearchUseCase,
		usecase.NewNodeSearchUseCase,
		usecase.NewEdgeSearchUseCase,
		usecase.NewCommunitySearchUseCase,
		
		searchgrpc.NewSearchServiceServer,
		server.NewGRPCServer,
		event.NewNatsSubscriber,
	)
	return nil, nil, nil
}
