package write

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
	CreateNodeWithOutbox(node NodeRecord, event OutboxEvent) error
	CreateNodeBundle(node NodeRecord, rels []RelationshipRecord, event OutboxEvent) error
	UpdateNodeWithOutbox(node NodeRecord, event OutboxEvent) error
	SoftDeleteNodeWithOutbox(node NodeRecord, event OutboxEvent) error
	CreateRelationshipWithOutbox(rel RelationshipRecord, event OutboxEvent) error
}

// Repository combines the source-of-truth read/write and outbox boundaries.
type Repository interface {
	Reader
	OutboxReader
	Writer
}
