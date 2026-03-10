package batch

import "github.com/google/wire"

var ProviderSet = wire.NewSet(NewNeo4jWriter, NewSemanticDeduper, NewQdrantIndexer, NewUsecaseWithIndexer)
