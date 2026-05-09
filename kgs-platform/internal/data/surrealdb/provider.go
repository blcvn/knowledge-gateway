package surrealdb

import "github.com/google/wire"

// ProviderSet is the Wire provider set for all SurrealDB adapters.
// Use this as a drop-in replacement for data.ProviderSet when storage_mode = "surrealdb".
var ProviderSet = wire.NewSet(
	NewSurrealGraphRepo,
	NewSurrealGraphWriteRepo,
	NewSurrealEntityReader,
	NewSurrealRegistryRepo,
	NewSurrealOntologyRepo,
	NewSurrealRulesRepo,
	NewSurrealPolicyRepo,
	NewSurrealLockManager,
	NewSurrealOverlayStore,
	NewSurrealVectorRetriever,
	NewSurrealTextRetriever,
	NewSurrealCentralityScorer,
	NewSurrealAnalyticsExecutor,
)
