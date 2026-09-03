package bootstrap

func InitObserveSearch(reg *bus.InProcessRegistry, db *pgxpool.Pool, cfg *config.Config) {
    bm25   := pkgsearch.NewBM25Index()
    vector := pkgsearch.NewVectorIndex(cfg.Search.EmbedDims)

    var embedder port.IEmbedder = &bifrostadapter.NullEmbedder{}
    if cfg.Search.EmbeddingProvider != "none" && cfg.Bifrost.URL != "" {
        embedder = bifrostadapter.NewBifrostEmbedder(cfg.Bifrost.URL, cfg.Search.EmbeddingModel, cfg.Search.EmbedDims)
    }

    persister := pkgsearch.NewIndexPersister(bm25, vector, cfg.Search.DataDir)
    persister.LoadAsync()

    obsStore   := postgresadapter.NewObservationStore(db)
    memClient  := grpcclient.NewAgentMemoryClient(reg)

    smartSearchUC  := usecase.NewSmartSearch(bm25, vector, embedder, pkgsearch.DefaultWeights, obsStore)
    buildContextUC := usecase.NewBuildContext(obsStore, memClient, smartSearchUC)
    indexAddUC     := usecase.NewIndexAdd(bm25, vector, embedder, persister)
    indexRemoveUC  := usecase.NewIndexRemove(bm25, vector, persister)

    handler := grpchandler.NewObserveSearchHandler(smartSearchUC, buildContextUC, indexAddUC, bm25, vector)

    grpcServer := grpc.NewServer()
    searchpb.RegisterObserveSearchServiceServer(grpcServer, handler)
    reg.Register("am-search", grpcServer)
}
