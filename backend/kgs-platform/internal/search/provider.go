package search

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewEmbeddingClient,
	NewVectorSearcher,
	NewTextSearcher,
	NewNeo4jCentralityProvider,
	NewEngine,
	wire.Bind(new(SearchEngine), new(*Engine)),
)
