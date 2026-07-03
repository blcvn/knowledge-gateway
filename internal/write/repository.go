package write

import "context"
import "time"

// Reader exposes source-of-truth lookup operations used by read, search, workers, and integrity code.
type Reader interface {
	GetNodeByID(id string) (NodeRecord, bool)
	GetNodesByIDs(ids []string) map[string]NodeRecord
	GetNodeByExternalRef(externalRef string) (NodeRecord, bool)
	GetNodesByExternalRefs(externalRefs []string) map[string]NodeRecord
	ListNodesBatch(afterID string, limit int) []NodeRecord
	ListNodes() []NodeRecord
	GetRelationshipByID(id string) (RelationshipRecord, bool)
	GetRelationshipsByIDs(ids []string) map[string]RelationshipRecord
	ListRelationshipsBatch(afterID string, limit int) []RelationshipRecord
	ListRelationships() []RelationshipRecord
}

// OutboxReader exposes the durable sync event stream.
type OutboxReader interface {
	ClaimOutboxBatch(ctx context.Context, pageSize int) ([]OutboxEvent, error)
	ListOutboxEvents() []OutboxEvent
}

// Writer exposes the mutation operations that must persist source data and outbox entries together.
type Writer interface {
	CreateNodesBulkWithOutbox(ctx context.Context, nodes []NodeRecord, events []OutboxEvent) error
	CreateNodeWithOutbox(ctx context.Context, node NodeRecord, event OutboxEvent) error
	CreateNodeBundle(ctx context.Context, node NodeRecord, rels []RelationshipRecord, event OutboxEvent) error
	CreateRelationshipsBulkWithOutbox(ctx context.Context, rels []RelationshipRecord, events []OutboxEvent) error
	UpdateNode(ctx context.Context, node NodeRecord) error
	UpdateNodeWithOutbox(ctx context.Context, node NodeRecord, event OutboxEvent) error
	SoftDeleteNode(ctx context.Context, node NodeRecord) error
	SoftDeleteNodeWithOutbox(ctx context.Context, node NodeRecord, event OutboxEvent) error
	SoftDeleteNodesByExternalRefPrefix(ctx context.Context, prefix string, deletedAt time.Time) ([]NodeRecord, error)
	SoftDeleteNodesByExternalRefPrefixWithOutbox(ctx context.Context, prefix string, deletedAt time.Time) ([]NodeRecord, error)
	CreateRelationshipWithOutbox(ctx context.Context, rel RelationshipRecord, event OutboxEvent) error
	SoftDeleteRelationshipsWithOutbox(ctx context.Context, relationshipIDs []string, deletedAt time.Time) ([]RelationshipRecord, error)
	CreateOutboxEvents(ctx context.Context, events []OutboxEvent) error
	UpdateOutboxStatus(ctx context.Context, eventID, status string, retryCount int, processedAt *time.Time) error
	UpsertProjectionVersion(ctx context.Context, record ProjectionVersionRecord) error
	GetProjectionVersion(entityID, entityKind string) (ProjectionVersionRecord, bool)
	ListProjectionVersionsBatch(afterEntityKind, afterEntityID string, limit int) []ProjectionVersionRecord
	ListProjectionVersions() []ProjectionVersionRecord
	SealGraphVersion(ctx context.Context, request GraphVersionSealRequest) (GraphIdentityRecord, GraphVersionRecord, error)
	FinalizeGraphVersion(ctx context.Context, versionID string) (int64, error)
	AddGraphVersionEntities(ctx context.Context, versionID string, entities []GraphVersionEntityRecord) error
	GetGraphVersionEntities(versionID string) []GraphVersionEntityRecord
	AbandonGraphVersion(ctx context.Context, versionID string) error
	CleanupExpiredSyncSession(ctx context.Context, versionID string) error
	GetGraphVersionByID(ctx context.Context, versionID string) (GraphVersionRecord, bool)
	GetGraphIdentityByID(ctx context.Context, identifierID string) (GraphIdentityRecord, bool)
	AcquireScopeLease(ctx context.Context, ownerTenantID, ownerAppID, graphScope, versionID string, expiresAt time.Time) error
	ReleaseScopeLease(ctx context.Context, ownerTenantID, ownerAppID, graphScope, versionID string) error
	GetScopeLease(ctx context.Context, ownerTenantID, ownerAppID, graphScope string) (ScopeLeaseRecord, bool)
	UpsertGraphProjectionHead(ctx context.Context, record GraphProjectionHeadRecord) error
	GetGraphProjectionHead(identifierID, backendKind, backendName string) (GraphProjectionHeadRecord, bool)
}

// Repository combines the source-of-truth read/write and outbox boundaries.
type Repository interface {
	Reader
	OutboxReader
	Writer
}
