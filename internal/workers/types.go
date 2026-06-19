package workers

import (
	"time"

	"kg-service/internal/write"
)

type EventStatus string

const (
	EventPending    EventStatus = "PENDING"
	EventProcessing EventStatus = "PROCESSING"
	EventDone       EventStatus = "DONE"
	EventFailed     EventStatus = "FAILED"
	EventDeadLetter EventStatus = "DEAD_LETTER"
)

type EventEnvelope struct {
	Event      write.OutboxEvent
	Status     EventStatus
	RetryCount int
	Error      string
}

type GraphNode struct {
	ID            string         `json:"id"`
	NodeType      string         `json:"node_type"`
	DomainID      string         `json:"domain_id"`
	OwnerTenantID string         `json:"owner_tenant_id"`
	OwnerAppID    string         `json:"owner_app_id"`
	ACLVisibleTo  []string       `json:"acl_visible_to"`
	Visibility    string         `json:"visibility,omitempty"`
	StatusValue   string         `json:"status_value,omitempty"`
	IsDeleted     bool           `json:"is_deleted"`
	SyncVersion   int64          `json:"_kg_sync_version,omitempty"`
	Properties    map[string]any `json:"properties"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type GraphRelationship struct {
	ID          string         `json:"id"`
	RelType     string         `json:"rel_type"`
	FromNodeID  string         `json:"from_node_id"`
	ToNodeID    string         `json:"to_node_id"`
	DomainID    string         `json:"domain_id"`
	SyncVersion int64          `json:"_kg_sync_version,omitempty"`
	Properties  map[string]any `json:"properties"`
}

type VectorDocument struct {
	NodeID         string         `json:"node_id"`
	NodeType       string         `json:"node_type"`
	DomainID       string         `json:"domain_id"`
	OwnerTenantID  string         `json:"owner_tenant_id"`
	OwnerAppID     string         `json:"owner_app_id"`
	ACLVisibleTo   []string       `json:"acl_visible_to"`
	IsDeleted      bool           `json:"is_deleted"`
	StatusValue    string         `json:"status_value,omitempty"`
	AuthorityScore *int           `json:"authority_score,omitempty"`
	SyncVersion    int64          `json:"_kg_sync_version,omitempty"`
	DomainProps    map[string]any `json:"domain_props"`
	Embedding      []float64      `json:"embedding,omitempty"`
}

type GraphStore struct {
	Nodes map[string]GraphNode
	Rels  map[string]GraphRelationship
}

type VectorStore struct {
	Documents map[string]VectorDocument
}

func (s *GraphStore) SnapshotNodes() map[string]GraphNode {
	result := make(map[string]GraphNode, len(s.Nodes))
	for id, node := range s.Nodes {
		result[id] = node
	}
	return result
}

func (s *GraphStore) SnapshotRelationships() map[string]GraphRelationship {
	result := make(map[string]GraphRelationship, len(s.Rels))
	for id, rel := range s.Rels {
		result[id] = rel
	}
	return result
}

func (s *VectorStore) SnapshotDocuments() map[string]VectorDocument {
	result := make(map[string]VectorDocument, len(s.Documents))
	for id, doc := range s.Documents {
		result[id] = doc
	}
	return result
}

type WorkerReport struct {
	Processed  int
	Failed     int
	DeadLetter int
}

type ReconciliationIssue struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Details string `json:"details"`
}

type SyncLagClass string

const (
	SyncLagClassSynced   SyncLagClass = "SYNCED"
	SyncLagClassInFlight SyncLagClass = "IN_FLIGHT"
	SyncLagClassLagging  SyncLagClass = "LAGGING"
	SyncLagClassStuck    SyncLagClass = "STUCK"
)

type EntitySyncStatus struct {
	EntityID           string       `json:"entity_id"`
	EntityKind         string       `json:"entity_kind"`
	SourceVersion      int64        `json:"source_version"`
	GraphVersion       int64        `json:"graph_version"`
	GraphLagClass      SyncLagClass `json:"graph_lag_class"`
	LastGraphSyncedAt  time.Time    `json:"last_graph_synced_at,omitempty"`
	VectorVersion      int64        `json:"vector_version"`
	VectorLagClass     SyncLagClass `json:"vector_lag_class"`
	LastVectorSyncedAt time.Time    `json:"last_vector_synced_at,omitempty"`
}

type ReconciliationReport struct {
	GraphDriftCount             int                   `json:"graph_drift_count"`
	VectorDriftCount            int                   `json:"vector_drift_count"`
	ProjectionVersionDriftCount int                   `json:"projection_version_drift_count"`
	GraphLaggingCount           int                   `json:"graph_lagging_count"`
	VectorLaggingCount          int                   `json:"vector_lagging_count"`
	GraphInFlightCount          int                   `json:"graph_in_flight_count"`
	VectorInFlightCount         int                   `json:"vector_in_flight_count"`
	Issues                      []ReconciliationIssue `json:"issues"`
	Overall                     string                `json:"overall"`
}
