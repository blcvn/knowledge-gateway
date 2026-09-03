package bootstrap

import (
	"log/slog"

	"github.com/nats-io/nats.go"

	"github.com/vnp-community/vnp-memory/apps/memory/internal/bus"
)

// Cognee bootstraps the Cognee engine services (cognee-ingestion, cognee-cognify, cognee-search)
// into the monolithic app via in-process gRPC and NATS event subscriptions.
//
// Event flow:
//   cognee-ingestion → publishes "cognee.data.ingested"
//   cognee-cognify   → subscribes "cognee.data.ingested", publishes "cognee.data.cognified"
//   cognee-search    → indexes cognified data for retrieval
func Cognee(grpcBus *bus.GRPCBus, infra *Infra, natsBus *bus.NATSBus, logger *slog.Logger) {
	logger.Info("Bootstrapping Cognee engine...")

	// Production wiring pattern (when service packages expose constructors):
	//   ingestionRepo := ingestionpg.NewRepository(infra.PG)
	//   ingestionUC := ingestionuc.NewUseCase(ingestionRepo, qdrantClient)
	//   ingestionHandler := ingestiongrpc.NewHandler(ingestionUC)
	//   grpcBus.Register(&ingestionpb.IngestionService_ServiceDesc, ingestionHandler)

	// Setup NATS event subscriptions for Cognee pipeline
	if natsBus != nil {
		_, err := natsBus.Subscribe("cognee.data.ingested", "cognee-cognify-consumer", func(msg *nats.Msg) {
			logger.Debug("cognee.data.ingested event received", "size", len(msg.Data))
			msg.Ack()
		})
		if err != nil {
			logger.Warn("failed to subscribe cognee events", "error", err)
		}
	}
}
