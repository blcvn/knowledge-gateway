package outbox

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewOutboxWorker,
	NewNeo4jSyncer,
	NewQdrantSyncer,
	NewReconcileJob,
)
