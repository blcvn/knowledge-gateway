package bootstrap

import (
	"log/slog"

	"github.com/nats-io/nats.go"

	"github.com/vnp-community/vnp-memory/apps/memory/internal/bus"
)

// Graphiti bootstraps the Graphiti engine services into the monolithic app.
// Services: graphiti-ingestion, graphiti-search, graphiti-knowledge, graphiti-store.
//
// Event flow:
//   graphiti-ingestion → publishes "graphiti.episode.ingested"
//   graphiti-knowledge → subscribes and builds knowledge graph
//   graphiti-store     → persists graph data to Neo4j
//   graphiti-search    → queries graph for retrieval
func Graphiti(grpcBus *bus.GRPCBus, infra *Infra, natsBus *bus.NATSBus, logger *slog.Logger) {
	logger.Info("Bootstrapping Graphiti engine...")

	// Production wiring pattern:
	//   neo4jDriver := infra.Neo4j
	//   storeRepo := storeneo4j.NewRepository(neo4jDriver)
	//   storeUC := storeuc.NewUseCase(storeRepo)
	//   storeHandler := storegrpc.NewHandler(storeUC)
	//   grpcBus.Register(&storepb.StoreService_ServiceDesc, storeHandler)

	if natsBus != nil {
		_, err := natsBus.Subscribe("graphiti.episode.ingested", "graphiti-knowledge-consumer", func(msg *nats.Msg) {
			logger.Debug("graphiti.episode.ingested event received", "size", len(msg.Data))
			msg.Ack()
		})
		if err != nil {
			logger.Warn("failed to subscribe graphiti events", "error", err)
		}
	}
}
