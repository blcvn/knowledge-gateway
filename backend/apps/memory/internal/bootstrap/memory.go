package bootstrap

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/vnp-community/vnp-memory/apps/memory/internal/bus"
)

// AgentMemory bootstraps the AgentMemory engine into the monolithic app.
// Wires: am-observe, am-memory, am-search.
//
// Event flow:
//
//	am-observe → publishes "agentmemory.observation.captured", "agentmemory.session.started/ended"
//	am-memory  → subscribes to consolidate observations into memories
//	am-search  → subscribes to update BM25+vector index on new observations
func AgentMemory(grpcBus *bus.GRPCBus, infra *Infra, natsBus *bus.NATSBus, logger *slog.Logger) {
	logger.Info("Bootstrapping AgentMemory engine...")

	// Production wiring pattern (uncomment after proto-agentmemory runs):
	//
	// ── observe-service ───────────────────────────────────────────────────
	//   dedup        := observe.NewDedupMap()
	//   go dedup.StartCleanup(context.Background())
	//   streamBroker := observe.NewStreamBroker()
	//   privacyR     := privacy.NewRedactor()
	//
	//   observeSearchClient := httpclient.NewSearchClient(cfg.ObserveSearch.URL)
	//   observeKVStore      := observepg.NewKVStore(infra.PG)
	//   observePublisher    := natevent.NewPublisher(natsConn)
	//   pipeline := observe.NewPipeline(dedup, observeKVStore, observeSearchClient,
	//       observePublisher, streamBroker, privacyR, observe.DefaultPipelineConfig())
	//
	//   sessionRepo   := observepg.NewSessionRepo(infra.PG)
	//   obsRepo       := observepg.NewObservationRepo(infra.PG)
	//   observeUC     := observeuc.NewObserveUseCase(pipeline, sessionRepo, obsRepo)
	//   createSessUC  := observeuc.NewCreateSessionUseCase(sessionRepo, observePublisher)
	//   endSessUC     := observeuc.NewEndSessionUseCase(sessionRepo, obsRepo, observePublisher)
	//   observeH      := observegrpc.NewObserveHandler(observeUC, createSessUC, endSessUC, sessionRepo, streamBroker)
	//   grpcBus.Register(&observepb.ObserveService_ServiceDesc, observeH)
	//
	// ── memory-service (AgentMemory) ──────────────────────────────────────
	//   memRepo       := postgres.NewAgentMemoryRepo(infra.PG)
	//   slotsRepo     := postgres.NewSlotsRepo(infra.PG)
	//   searchNotifier := httpclient.NewSearchClient(cfg.ObserveSearch.URL)
	//   publisher      := natevent.NewPublisher(natsConn)
	//
	//   rememberUC     := agentmemory.NewRememberAgentUseCase(memRepo, searchNotifier, publisher)
	//   evictUC        := agentmemory.NewEvictUseCase(memRepo, publisher)
	//   autoForgetUC   := agentmemory.NewAutoForgetUseCase(memRepo, searchNotifier, publisher)
	//   retentionUC    := agentmemory.NewRetentionUseCase(memRepo)
	//   decayScheduler := agentmemory.NewDecayScheduler(memRepo, cfg.Memory.HalfLifeDays)
	//   slotsUC        := agentmemory.NewSlotsUseCase(slotsRepo)
	//
	//   agentMemH := memgrpc.NewAgentMemoryHandler(rememberUC, evictUC, autoForgetUC, retentionUC, slotsUC, memRepo)
	//   grpcBus.Register(&agentmemorypb.AgentMemoryService_ServiceDesc, agentMemH)
	//
	//   go autoForgetUC.StartScheduler(context.Background())
	//   go decayScheduler.Start(context.Background())

	if natsBus != nil {
		// am-memory subscribes to new observations to consolidate into long-term memory
		_, err := natsBus.Subscribe("agentmemory.observation.captured", "am-memory-consolidator", func(msg *nats.Msg) {
			logger.Debug("agentmemory.observation.captured received", "size", len(msg.Data))
			// In production: memoryConsolidationUC.Process(ctx, msg.Data)
			msg.Ack()
		})
		if err != nil {
			logger.Warn("failed to subscribe agentmemory observation events", "error", err)
		}

		// am-search subscribes to index new observations into BM25+vector
		_, err = natsBus.Subscribe("agentmemory.observation.captured", "am-search-indexer", func(msg *nats.Msg) {
			logger.Debug("agentmemory search indexer: observation received", "size", len(msg.Data))
			// In production: searchIndexUC.IndexObservation(ctx, msg.Data)
			msg.Ack()
		})
		if err != nil {
			logger.Warn("failed to subscribe agentmemory search events", "error", err)
		}

		// am-session events for lifecycle tracking
		_, err = natsBus.Subscribe("agentmemory.session.started", "am-session-tracker", func(msg *nats.Msg) {
			logger.Debug("agentmemory.session.started received", "size", len(msg.Data))
			msg.Ack()
		})
		if err != nil {
			logger.Warn("failed to subscribe agentmemory session.started events", "error", err)
		}

		_, err = natsBus.Subscribe("agentmemory.session.ended", "am-session-tracker", func(msg *nats.Msg) {
			logger.Debug("agentmemory.session.ended received", "size", len(msg.Data))
			msg.Ack()
		})
		if err != nil {
			logger.Warn("failed to subscribe agentmemory session.ended events", "error", err)
		}
	}

	// Ensure context is used if needed
	_ = context.Background()
}
