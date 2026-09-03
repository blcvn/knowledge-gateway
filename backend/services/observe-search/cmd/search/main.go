package main

import (
    "log"
    "os"

    "github.com/vnp-memory/services/observe-search/internal/adapter/bifrost"
    "github.com/vnp-memory/services/observe-search/internal/index"
    intsearch "github.com/vnp-memory/services/observe-search/internal/search"
    pkg_search "github.com/vnp-memory/pkg/search"
)

func main() {
    log.Println("observe-search service starting")

    dims := 384
    bm25 := pkg_search.NewBM25Index()
    vector := pkg_search.NewVectorIndex(dims)

    var embedder intsearch.IEmbedder = &bifrost.NullEmbedder{}
    if os.Getenv("EMBEDDING_PROVIDER") != "none" && os.Getenv("BIFROST_URL") != "" {
        embedder = bifrost.NewEmbedder(
            os.Getenv("BIFROST_URL"),
            os.Getenv("EMBEDDING_MODEL"),
            dims,
        )
    }

    persister := pkg_search.NewIndexPersister(bm25, vector, os.Getenv("SEARCH_DATA_DIR"))
    persister.LoadAsync()

    indexMgr := index.NewManager(bm25, vector, embedder, persister)
    _ = indexMgr

    weights := pkg_search.ScoreWeights{BM25: 0.5, Vector: 0.5}
    _ = intsearch.NewSmartSearch(bm25, vector, embedder, weights)

    // TODO: start gRPC server
    select {}
}
