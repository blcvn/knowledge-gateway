package bootstrap

import (
	"log/slog"

	"github.com/nats-io/nats.go"

	"github.com/vnp-community/vnp-memory/apps/memory/internal/bus"
)

// Memobase bootstraps the Memobase engine services into the monolithic app.
// Services: memobase-ingestion, memobase-engine, memobase-context.
//
// Event flow:
//   memobase-ingestion → buffers user blobs, publishes "memobase.buffer.flush"
//   memobase-engine    → subscribes to flush events, processes & consolidates
//   memobase-context   → serves context/profile queries
func Memobase(grpcBus *bus.GRPCBus, infra *Infra, natsBus *bus.NATSBus, logger *slog.Logger) {
	logger.Info("Bootstrapping Memobase engine...")

	// Production wiring pattern:
	//   ingestionRepo := ingestionpg.NewRepository(infra.PG)
	//   ingestionUC := ingestionuc.NewUseCase(ingestionRepo, natsBus.Conn())
	//   ingestionHandler := ingestiongrpc.NewHandler(ingestionUC)
	//   grpcBus.Register(&ingestionpb.IngestionService_ServiceDesc, ingestionHandler)

	if natsBus != nil {
		_, err := natsBus.Subscribe("memobase.buffer.flush", "memobase-engine-consumer", func(msg *nats.Msg) {
			logger.Debug("memobase.buffer.flush event received", "size", len(msg.Data))
			msg.Ack()
		})
		if err != nil {
			logger.Warn("failed to subscribe memobase flush events", "error", err)
		}

		_, err = natsBus.Subscribe("memobase.profile.updated", "memobase-context-consumer", func(msg *nats.Msg) {
			logger.Debug("memobase.profile.updated event received", "size", len(msg.Data))
			msg.Ack()
		})
		if err != nil {
			logger.Warn("failed to subscribe memobase profile events", "error", err)
		}
	}
}
