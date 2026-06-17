package workers

import "kg-service/internal/write"

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
	StatusValue   string         `json:"status_value,omitempty"`
	IsDeleted     bool           `json:"is_deleted"`
	Properties    map[string]any `json:"properties"`
}

type GraphRelationship struct {
	ID         string         `json:"id"`
	RelType    string         `json:"rel_type"`
	FromNodeID string         `json:"from_node_id"`
	ToNodeID   string         `json:"to_node_id"`
	DomainID   string         `json:"domain_id"`
	Properties map[string]any `json:"properties"`
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

type ReconciliationReport struct {
	GraphDriftCount  int                   `json:"graph_drift_count"`
	VectorDriftCount int                   `json:"vector_drift_count"`
	Issues           []ReconciliationIssue `json:"issues"`
	Overall          string                `json:"overall"`
}
