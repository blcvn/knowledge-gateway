package bootstrap

import (
	"context"
	"log/slog"

	"github.com/vnp-community/vnp-memory/apps/memory/internal/bus"
	"github.com/vnp-community/vnp-memory/apps/memory/internal/config"
	"github.com/vnp-community/vnp-memory/shared/pkg/forward"
	dashboardHandler "github.com/vnp-community/vnp-memory/services/vnp-dashboard/adapter/grpc"
	searchHandler "github.com/vnp-memory/services/vnp-search-hub/adapter/grpc"
	graphitiHandler "github.com/vnp-memory/services/graphiti-store/adapter/grpc"
)

// wrapHandler wraps a simple context handler into a forward.HandlerFunc.
func wrapHandler(fn func(ctx context.Context) ([]byte, error)) forward.HandlerFunc {
	return func(ctx context.Context, body []byte, params map[string]string) ([]byte, error) {
		return fn(ctx)
	}
}

// Platform configures core platform components like UI handlers, search orchestrators,
// and other embedded monolith engines.
func Platform(grpcBus *bus.GRPCBus, infra *Infra, natsBus *bus.NATSBus, cfg *config.Config, logger *slog.Logger) {
	logger.Info("Bootstrapping Platform services...")

	// Wire VNP Dashboard Service
	dH := dashboardHandler.NewDashboardHandler()

	// Setup ForwardService router for console APIs
	router := forward.NewRouter(logger)
	router.Handle("GET", "/v1/console/dashboard/health", wrapHandler(dH.GetHealth))
	router.Handle("GET", "/v1/console/dashboard/metrics", wrapHandler(dH.GetMetrics))
	router.Handle("GET", "/v1/console/dashboard/throughput", wrapHandler(dH.GetThroughput))
	router.Handle("GET", "/v1/console/dashboard/heatmap", wrapHandler(dH.GetHeatmap))

	// Wire vnp-search-hub
	sH := searchHandler.NewConsoleSearchHandler(nil, logger)
	router.Handle("POST", "/v1/console/memory/search", wrapHandler(sH.HandleSearch))
	router.Handle("GET", "/v1/console/memory/{id}", wrapHandler(sH.HandleGetMemory))
	router.Handle("GET", "/v1/console/memory/{id}/neighbors", wrapHandler(sH.HandleGetNeighbors))
	router.Handle("POST", "/v1/console/debugger/trace", wrapHandler(sH.HandleCreateTrace))

	// Wire graphiti-store
	gH := graphitiHandler.NewConsoleGraphHandler()
	router.Handle("POST", "/v1/console/graph/subgraph", wrapHandler(gH.HandleSubgraph))
	router.Handle("GET", "/v1/console/graph/entity/{id}", wrapHandler(gH.HandleGetEntity))
	router.Handle("POST", "/v1/console/graph/timeline", wrapHandler(gH.HandleTimeline))
	router.Handle("POST", "/v1/console/graph/query", wrapHandler(gH.HandleQuery))

	// Register router as a gRPC service
	forward.RegisterForwardService(grpcBus.Server(), router)

	// Mark platform services as registered so InProcessRegistry routes to them
	grpcBus.RegisterServiceMarker("vnp-dashboard")
	grpcBus.RegisterServiceMarker("vnp-search-hub")
	grpcBus.RegisterServiceMarker("graphiti-store")
	
	// We could wire vnp-admin, vnp-event here too as needed.
}
