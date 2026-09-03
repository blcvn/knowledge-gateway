package bootstrap

import (
	"log/slog"

	"github.com/nats-io/nats.go"

	"github.com/vnp-community/vnp-memory/apps/memory/internal/bus"
)

// Supermemory bootstraps the Supermemory engine services into the monolithic app.
// Services: sm-document, sm-memory, sm-search, sm-profile, sm-connector,
//           sm-mcp, sm-auth, sm-analytics, sm-project.
//
// Supermemory provides adaptive memory with forgetting curves, document management,
// external connectors, and per-project analytics.
//
// Event flow:
//   sm-document  → publishes "sm.document.created" after document ingestion
//   sm-memory    → subscribes to process and store adaptive memories
//   sm-search    → subscribes to update search index
//   sm-analytics → subscribes to track usage patterns
func Supermemory(grpcBus *bus.GRPCBus, infra *Infra, natsBus *bus.NATSBus, logger *slog.Logger) {
	logger.Info("Bootstrapping Supermemory engine...")

	// Production wiring pattern:
	//   documentRepo := documentpg.NewRepository(infra.PG)
	//   documentUC := documentuc.NewUseCase(documentRepo, infra.Qdrant)
	//   documentHandler := documentgrpc.NewHandler(documentUC)
	//   grpcBus.Register(&documentpb.DocumentService_ServiceDesc, documentHandler)
	//
	//   memoryRepo := memorypg.NewRepository(infra.PG)
	//   memoryUC := memoryuc.NewUseCase(memoryRepo, infra.Qdrant)
	//   memoryHandler := memorygrpc.NewHandler(memoryUC)
	//   grpcBus.Register(&memorypb.MemoryService_ServiceDesc, memoryHandler)
	//
	//   searchUC := searchuc.NewUseCase(infra.Qdrant, documentRepo, memoryRepo)
	//   searchHandler := searchgrpc.NewHandler(searchUC)
	//   grpcBus.Register(&searchpb.SearchService_ServiceDesc, searchHandler)
	//
	//   profileRepo := profilepg.NewRepository(infra.PG)
	//   profileUC := profileuc.NewUseCase(profileRepo)
	//   profileHandler := profilegrpc.NewHandler(profileUC)
	//   grpcBus.Register(&profilepb.ProfileService_ServiceDesc, profileHandler)
	//
	//   connectorRepo := connectorpg.NewRepository(infra.PG)
	//   connectorUC := connectoruc.NewUseCase(connectorRepo)
	//   connectorHandler := connectorgrpc.NewHandler(connectorUC)
	//   grpcBus.Register(&connectorpb.ConnectorService_ServiceDesc, connectorHandler)
	//
	//   mcpUC := mcpuc.NewUseCase(documentRepo, memoryRepo)
	//   mcpHandler := mcpgrpc.NewHandler(mcpUC)
	//   grpcBus.Register(&mcppb.MCPService_ServiceDesc, mcpHandler)
	//
	//   authUC := authuc.NewUseCase(profileRepo)
	//   authHandler := authgrpc.NewHandler(authUC)
	//   grpcBus.Register(&authpb.AuthService_ServiceDesc, authHandler)
	//
	//   analyticsRepo := analyticspg.NewRepository(infra.PG)
	//   analyticsUC := analyticsuc.NewUseCase(analyticsRepo)
	//   analyticsHandler := analyticsgrpc.NewHandler(analyticsUC)
	//   grpcBus.Register(&analyticspb.AnalyticsService_ServiceDesc, analyticsHandler)
	//
	//   projectRepo := projectpg.NewRepository(infra.PG)
	//   projectUC := projectuc.NewUseCase(projectRepo)
	//   projectHandler := projectgrpc.NewHandler(projectUC)
	//   grpcBus.Register(&projectpb.ProjectService_ServiceDesc, projectHandler)

	if natsBus != nil {
		// sm-memory subscribes to document creation events
		_, err := natsBus.Subscribe("sm.document.created", "sm-memory-processor", func(msg *nats.Msg) {
			logger.Debug("sm.document.created event received", "size", len(msg.Data))
			// In production: memoryUC.ProcessDocument(ctx, msg.Data)
			msg.Ack()
		})
		if err != nil {
			logger.Warn("failed to subscribe sm document events", "error", err)
		}

		// sm-search subscribes to memory update events for index refresh
		_, err = natsBus.Subscribe("sm.memory.updated", "sm-search-indexer", func(msg *nats.Msg) {
			logger.Debug("sm.memory.updated event received", "size", len(msg.Data))
			// In production: searchUC.IndexMemory(ctx, msg.Data)
			msg.Ack()
		})
		if err != nil {
			logger.Warn("failed to subscribe sm memory events", "error", err)
		}

		// sm-analytics subscribes to track usage across all operations
		_, err = natsBus.Subscribe("sm.analytics.event", "sm-analytics-tracker", func(msg *nats.Msg) {
			logger.Debug("sm.analytics.event received", "size", len(msg.Data))
			// In production: analyticsUC.Track(ctx, msg.Data)
			msg.Ack()
		})
		if err != nil {
			logger.Warn("failed to subscribe sm analytics events", "error", err)
		}
	}
}
