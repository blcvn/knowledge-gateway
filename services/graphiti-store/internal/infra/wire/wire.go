//go:build wireinject
// +build wireinject

package wire

import (
	"log/slog"
	"os"

	"github.com/google/wire"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/adapter/factory"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/adapter/grpc"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/infra/config"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/infra/server"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/usecase"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/usecase/port"
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
