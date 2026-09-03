//go:build wireinject
// +build wireinject

package wire

import (
	"log/slog"
	"os"

	"github.com/google/wire"
	"vnp-memory/services/graphiti-store/adapter/factory"
	"vnp-memory/services/graphiti-store/adapter/grpc"
	"vnp-memory/services/graphiti-store/infra/config"
	"vnp-memory/services/graphiti-store/infra/server"
	"vnp-memory/services/graphiti-store/usecase"
	"vnp-memory/services/graphiti-store/usecase/port"
)

func ProvideLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

var usecaseProvider = wire.NewSet(
	usecase.NewNodeService,
	wire.Bind(new(port.NodeService), new(*usecase.NodeServiceImpl)),
	usecase.NewEdgeService,
	wire.Bind(new(port.EdgeService), new(*usecase.EdgeServiceImpl)),
	usecase.NewSearchService,
	wire.Bind(new(port.SearchService), new(*usecase.SearchServiceImpl)),
	usecase.NewBulkService,
	wire.Bind(new(port.BulkService), new(*usecase.BulkServiceImpl)),
	// Default vector dim is passed explicitly or read from config. Assuming 1536 for now.
	wire.Value(1536),
	usecase.NewIndexService,
	wire.Bind(new(port.IndexService), new(*usecase.IndexServiceImpl)),
	usecase.NewCommunityService,
	wire.Bind(new(port.CommunityService), new(*usecase.CommunityServiceImpl)),
)

// InitializeServer sets up the entire application dependency graph.
func InitializeServer(cfg config.Config) (*server.GRPCServer, error) {
	wire.Build(
		ProvideLogger,
		factory.NewGraphDriver,
		usecaseProvider,
		grpc.NewHandler,
		server.NewGRPCServer,
	)
	return nil, nil
}
