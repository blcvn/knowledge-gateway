package bootstrap

func InitOrchestration(reg *bus.InProcessRegistry, db *pgxpool.Pool, nc *nats.Conn, cfg *config.Config) {
    repos := orchrepo.NewPostgresRepos(db)
    publisher := natevent.NewPublisher(nc, "agentmemory")

    var llm port.ILLMProvider
    if cfg.Bifrost.URL != "" {
        llm = bifrost.NewLLMClient(cfg.Bifrost.URL)
    }

    actionSvc     := orchestration.NewActionService(repos.Actions, publisher)
    leaseSvc      := orchestration.NewLeaseService(repos.Leases, publisher)
    signalSvc     := orchestration.NewSignalService(repos.Signals, publisher)
    checkpointSvc := orchestration.NewCheckpointService(repos.Checkpoints, publisher)
    routineSvc    := orchestration.NewRoutineService(repos.Routines)
    sentinelSvc   := orchestration.NewSentinelService(repos.Sentinels, repos.Actions, publisher)
    sketchSvc     := orchestration.NewSketchService(repos.Sketches, repos.Actions, repos.Crystals, llm)

    handler := grpchandler.NewOrchestrationHandler(
        actionSvc, leaseSvc, signalSvc, checkpointSvc, routineSvc, sentinelSvc, sketchSvc,
        repos.Actions, repos.Signals, repos.Crystals,
    )

    grpcServer := grpc.NewServer()
    orchpb.RegisterOrchestrationServiceServer(grpcServer, handler)
    reg.Register("am-orchestration", grpcServer)

    sweeper := background.NewBackgroundSweeper(leaseSvc, signalSvc, sentinelSvc, sketchSvc, checkpointSvc)
    go sweeper.Start(context.Background())
}
