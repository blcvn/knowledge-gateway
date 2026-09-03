package bootstrap

import (
	"log/slog"

	"github.com/nats-io/nats.go"

	"github.com/vnp-community/vnp-memory/apps/memory/internal/bus"
)

// OpenViking bootstraps the OpenViking engine services into the monolithic app.
// Services: ov-fs, ov-search, ov-session, ov-resource, ov-crypto, ov-admin.
//
// OpenViking provides a hierarchical context database with file system semantics,
// encrypted storage, and session-based editing workflows.
//
// Event flow:
//   ov-resource → publishes "ov.resource.ingested" after new content ingestion
//   ov-search   → subscribes to update its search index
//   ov-session  → manages editing sessions and commit lifecycle
func OpenViking(grpcBus *bus.GRPCBus, infra *Infra, natsBus *bus.NATSBus, logger *slog.Logger) {
	logger.Info("Bootstrapping OpenViking engine...")

	// Production wiring pattern:
	//   fsRepo := fspg.NewRepository(infra.PG)
	//   fsUC := fsuc.NewUseCase(fsRepo, infra.MinIO)
	//   fsHandler := fsgrpc.NewHandler(fsUC)
	//   grpcBus.Register(&fspb.FSService_ServiceDesc, fsHandler)
	//
	//   searchRepo := searchpg.NewRepository(infra.PG)
	//   searchUC := searchuc.NewUseCase(searchRepo, infra.Qdrant)
	//   searchHandler := searchgrpc.NewHandler(searchUC)
	//   grpcBus.Register(&searchpb.SearchService_ServiceDesc, searchHandler)
	//
	//   sessionRepo := sessionpg.NewRepository(infra.PG)
	//   sessionUC := sessionuc.NewUseCase(sessionRepo)
	//   sessionHandler := sessiongrpc.NewHandler(sessionUC)
	//   grpcBus.Register(&sessionpb.SessionService_ServiceDesc, sessionHandler)
	//
	//   resourceUC := resourceuc.NewUseCase(fsRepo, searchRepo, natsBus)
	//   resourceHandler := resourcegrpc.NewHandler(resourceUC)
	//   grpcBus.Register(&resourcepb.ResourceService_ServiceDesc, resourceHandler)
	//
	//   cryptoUC := cryptouc.NewUseCase()
	//   cryptoHandler := cryptogrpc.NewHandler(cryptoUC)
	//   grpcBus.Register(&cryptopb.CryptoService_ServiceDesc, cryptoHandler)
	//
	//   adminUC := adminuc.NewUseCase(fsRepo)
	//   adminHandler := admingrpc.NewHandler(adminUC)
	//   grpcBus.Register(&adminpb.AdminService_ServiceDesc, adminHandler)

	if natsBus != nil {
		// ov-search subscribes to resource ingestion events to update index
		_, err := natsBus.Subscribe("ov.resource.ingested", "ov-search-indexer", func(msg *nats.Msg) {
			logger.Debug("ov.resource.ingested event received", "size", len(msg.Data))
			// In production: searchUC.IndexResource(ctx, msg.Data)
			msg.Ack()
		})
		if err != nil {
			logger.Warn("failed to subscribe ov resource events", "error", err)
		}

		// ov-session subscribes to session commit events
		_, err = natsBus.Subscribe("ov.session.committed", "ov-resource-sync", func(msg *nats.Msg) {
			logger.Debug("ov.session.committed event received", "size", len(msg.Data))
			// In production: resourceUC.SyncFromSession(ctx, msg.Data)
			msg.Ack()
		})
		if err != nil {
			logger.Warn("failed to subscribe ov session events", "error", err)
		}
	}
}
