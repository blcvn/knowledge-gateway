package bootstrap

import (
	"log/slog"

	"github.com/nats-io/nats.go"

	"github.com/vnp-community/vnp-memory/apps/memory/internal/bus"
)

// Zep bootstraps the Zep engine services into the monolithic app.
// Services: zep-user, zep-thread, zep-memory, zep-graph, zep-search, zep-admin.
//
// Zep provides conversational memory with fact extraction, graph-based knowledge,
// and context assembly for LLM interactions.
//
// Event flow:
//   zep-memory → publishes "zep.memory.messages.ingested" after message processing
//   zep-graph  → subscribes to extract facts and build knowledge graph
//   zep-search → subscribes to update semantic search index
func Zep(grpcBus *bus.GRPCBus, infra *Infra, natsBus *bus.NATSBus, logger *slog.Logger) {
	logger.Info("Bootstrapping Zep engine...")

	// Production wiring pattern:
	//   userRepo := userpg.NewRepository(infra.PG)
	//   userUC := useruc.NewUseCase(userRepo)
	//   userHandler := usergrpc.NewHandler(userUC)
	//   grpcBus.Register(&userpb.UserService_ServiceDesc, userHandler)
	//
	//   threadRepo := threadpg.NewRepository(infra.PG)
	//   threadUC := threaduc.NewUseCase(threadRepo)
	//   threadHandler := threadgrpc.NewHandler(threadUC)
	//   grpcBus.Register(&threadpb.ThreadService_ServiceDesc, threadHandler)
	//
	//   memoryRepo := memorypg.NewRepository(infra.PG)
	//   memoryUC := memoryuc.NewUseCase(memoryRepo, infra.Qdrant, natsBus)
	//   memoryHandler := memorygrpc.NewHandler(memoryUC)
	//   grpcBus.Register(&memorypb.MemoryService_ServiceDesc, memoryHandler)
	//
	//   graphRepo := graphneo4j.NewRepository(infra.Neo4j)
	//   graphUC := graphuc.NewUseCase(graphRepo)
	//   graphHandler := graphgrpc.NewHandler(graphUC)
	//   grpcBus.Register(&graphpb.GraphService_ServiceDesc, graphHandler)
	//
	//   searchUC := searchuc.NewUseCase(infra.Qdrant, memoryRepo)
	//   searchHandler := searchgrpc.NewHandler(searchUC)
	//   grpcBus.Register(&searchpb.SearchService_ServiceDesc, searchHandler)
	//
	//   adminUC := adminuc.NewUseCase(userRepo, threadRepo, memoryRepo)
	//   adminHandler := admingrpc.NewHandler(adminUC)
	//   grpcBus.Register(&adminpb.AdminService_ServiceDesc, adminHandler)

	if natsBus != nil {
		// zep-graph subscribes to memory message events for fact extraction
		_, err := natsBus.Subscribe("zep.memory.messages.ingested", "zep-graph-extractor", func(msg *nats.Msg) {
			logger.Debug("zep.memory.messages.ingested event received", "size", len(msg.Data))
			// In production: graphUC.ExtractFacts(ctx, msg.Data)
			msg.Ack()
		})
		if err != nil {
			logger.Warn("failed to subscribe zep memory events", "error", err)
		}

		// zep-search subscribes to build search index from new messages
		_, err = natsBus.Subscribe("zep.memory.indexed", "zep-search-indexer", func(msg *nats.Msg) {
			logger.Debug("zep.memory.indexed event received", "size", len(msg.Data))
			// In production: searchUC.IndexMemory(ctx, msg.Data)
			msg.Ack()
		})
		if err != nil {
			logger.Warn("failed to subscribe zep index events", "error", err)
		}
	}
}
