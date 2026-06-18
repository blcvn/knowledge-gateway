package write

import "context"
import "time"

// Reader exposes source-of-truth lookup operations used by read, search, workers, and integrity code.
type Reader interface {
	GetNodeByID(id string) (NodeRecord, bool)
	GetNodeByExternalRef(externalRef string) (NodeRecord, bool)
	ListNodes() []NodeRecord
	GetRelationshipByID(id string) (RelationshipRecord, bool)
	ListRelationships() []RelationshipRecord
}

// OutboxReader exposes the durable sync event stream.
type OutboxReader interface {
	ListOutboxEvents() []OutboxEvent
}

// Writer exposes the mutation operations that must persist source data and outbox entries together.
type Writer interface {
	CreateNodeWithOutbox(ctx context.Context, node NodeRecord, event OutboxEvent) error
	CreateNodeBundle(ctx context.Context, node NodeRecord, rels []RelationshipRecord, event OutboxEvent) error
	UpdateNodeWithOutbox(ctx context.Context, node NodeRecord, event OutboxEvent) error
	SoftDeleteNodeWithOutbox(ctx context.Context, node NodeRecord, event OutboxEvent) error
	CreateRelationshipWithOutbox(ctx context.Context, rel RelationshipRecord, event OutboxEvent) error
	UpdateOutboxStatus(ctx context.Context, eventID, status string, retryCount int, processedAt *time.Time) error
	UpsertProjectionVersion(ctx context.Context, record ProjectionVersionRecord) error
	GetProjectionVersion(entityID, entityKind string) (ProjectionVersionRecord, bool)
	ListProjectionVersions() []ProjectionVersionRecord
}

// Repository combines the source-of-truth read/write and outbox boundaries.
type Repository interface {
	Reader
	OutboxReader
	Writer
}
