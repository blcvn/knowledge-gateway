package write

import "time"

type NodeCreateRequest struct {
	DomainID      string         `json:"domain_id"`
	NodeType      string         `json:"node_type"`
	Properties    map[string]any `json:"properties"`
	Visibility    string         `json:"visibility"`
	ExternalRef   string         `json:"external_ref"`
	ReferenceID   string         `json:"reference_id,omitempty"`
	BridgeNodeIDs []string       `json:"bridge_node_ids,omitempty"`
}

type NodeBulkCreateRequest struct {
	Nodes          []NodeCreateRequest `json:"nodes"`
	GraphVersionID string              `json:"graph_version_id,omitempty"`
}

type NodeBulkCreateResponse struct {
	Succeeded []NodeCreateResponse `json:"succeeded"`
	Failed    []BulkItemError      `json:"failed"`
}

type NodeRecord struct {
	ID            string         `json:"id"`
	NodeType      string         `json:"node_type"`
	DomainID      string         `json:"domain_id"`
	OwnerTenantID string         `json:"owner_tenant_id"`
	OwnerAppID    string         `json:"owner_app_id"`
	ACLVisibleTo  []string       `json:"acl_visible_to,omitempty"`
	Visibility    string         `json:"visibility"`
	Properties    map[string]any `json:"properties"`
	DomainVersion int            `json:"domain_version"`
	ExternalRef   string         `json:"external_ref,omitempty"`
	StatusValue   string         `json:"status_value,omitempty"`
	IsDeleted     bool           `json:"is_deleted"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type OutboxEvent struct {
	ID            string         `json:"id"`
	AggregateType string         `json:"aggregate_type"`
	AggregateID   string         `json:"aggregate_id"`
	EventType     string         `json:"event_type"`
	Payload       map[string]any `json:"payload"`
	Status        string         `json:"status"`
	RetryCount    int            `json:"retry_count"`
	CreatedAt     time.Time      `json:"created_at"`
	ProcessedAt   *time.Time     `json:"processed_at,omitempty"`
}

type GraphIdentityRecord struct {
	IdentifierID      string    `json:"identifier_id"`
	OwnerTenantID     string    `json:"owner_tenant_id"`
	OwnerAppID        string    `json:"owner_app_id"`
	GraphScope        string    `json:"graph_scope"`
	HeadVersionNumber int64     `json:"head_version_number"`
	HeadVersionID     string    `json:"head_version_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ScopeLeaseRecord struct {
	OwnerTenantID string    `json:"owner_tenant_id"`
	OwnerAppID    string    `json:"owner_app_id"`
	GraphScope    string    `json:"graph_scope"`
	VersionID     string    `json:"version_id"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type GraphVersionEntityRecord struct {
	VersionID  string `json:"version_id"`
	EntityKind string `json:"entity_kind"`
	EntityID   string `json:"entity_id"`
	ChangeKind string `json:"change_kind"`
}

type GraphVersionRecord struct {
	VersionID       string    `json:"version_id"`
	IdentifierID    string    `json:"identifier_id"`
	VersionNumber   int64     `json:"version_number"`
	ReferenceID     string    `json:"reference_id"`
	StorageClass    string    `json:"storage_class"`
	VersionStatus   string    `json:"version_status"`
	ChangeSummary   string    `json:"change_summary,omitempty"`
	ManifestLocator string    `json:"manifest_locator,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	SealedAt        time.Time `json:"sealed_at"`
}

type GraphProjectionHeadRecord struct {
	IdentifierID         string    `json:"identifier_id"`
	BackendKind          string    `json:"backend_kind"`
	BackendName          string    `json:"backend_name"`
	AppliedVersionID     string    `json:"applied_version_id,omitempty"`
	AppliedVersionNumber int64     `json:"applied_version_number"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type GraphVersionSealRequest struct {
	OwnerTenantID   string
	OwnerAppID      string
	GraphScope      string
	ReferenceID     string
	StorageClass    string
	VersionStatus   string
	ChangeSummary   string
	ManifestLocator string
	Entities        []GraphVersionEntityRecord
}

type RelationshipCreateRequest struct {
	RelType     string         `json:"rel_type"`
	FromNodeID  string         `json:"from_node_id"`
	ToNodeID    string         `json:"to_node_id"`
	DomainID    string         `json:"domain_id"`
	ReferenceID string         `json:"reference_id,omitempty"`
	Properties  map[string]any `json:"properties"`
}

type RelationshipBulkCreateRequest struct {
	Relationships  []RelationshipCreateRequest `json:"relationships"`
	GraphVersionID string                      `json:"graph_version_id,omitempty"`
}

type RelationshipBulkCreateResponse struct {
	Succeeded []RelationshipCreateResponse `json:"succeeded"`
	Failed    []BulkItemError              `json:"failed"`
}

type RelationshipRecord struct {
	ID            string         `json:"id"`
	RelType       string         `json:"rel_type"`
	FromNodeID    string         `json:"from_node_id"`
	ToNodeID      string         `json:"to_node_id"`
	DomainID      string         `json:"domain_id"`
	OwnerTenantID string         `json:"owner_tenant_id"`
	OwnerAppID    string         `json:"owner_app_id"`
	DomainVersion int            `json:"domain_version"`
	Properties    map[string]any `json:"properties"`
	IsDeleted     bool           `json:"is_deleted"`
	CreatedAt     time.Time      `json:"created_at"`
}

type ProjectionVersionRecord struct {
	EntityID           string    `json:"entity_id"`
	EntityKind         string    `json:"entity_kind"`
	SourceVersion      int64     `json:"source_version"`
	SourceEventID      string    `json:"source_event_id"`
	SourceUpdatedAt    time.Time `json:"source_updated_at"`
	GraphBackend       string    `json:"graph_backend"`
	GraphVersion       int64     `json:"graph_version"`
	LastGraphSyncedAt  time.Time `json:"last_graph_synced_at"`
	VectorBackend      string    `json:"vector_backend"`
	VectorVersion      int64     `json:"vector_version"`
	LastVectorSyncedAt time.Time `json:"last_vector_synced_at"`
}

type RelationshipCreateResponse struct {
	RelationshipID     string `json:"relationship_id"`
	GraphIdentifierID  string `json:"graph_identifier_id,omitempty"`
	GraphVersionID     string `json:"graph_version_id,omitempty"`
	GraphVersionNumber int64  `json:"graph_version_number,omitempty"`
	ReferenceID        string `json:"reference_id,omitempty"`
	Status             string `json:"status"`
}

type NodeCreateResponse struct {
	NodeID             string `json:"node_id"`
	GraphIdentifierID  string `json:"graph_identifier_id,omitempty"`
	GraphVersionID     string `json:"graph_version_id,omitempty"`
	GraphVersionNumber int64  `json:"graph_version_number,omitempty"`
	ReferenceID        string `json:"reference_id,omitempty"`
	DomainVersion      int    `json:"domain_version"`
	Status             string `json:"status"`
	SyncETAMs          int    `json:"sync_eta_ms"`
}

type NodeUpdateRequest struct {
	Properties     map[string]any `json:"properties"`
	Visibility     string         `json:"visibility"`
	ExternalRef    string         `json:"external_ref"`
	ReferenceID    string         `json:"reference_id,omitempty"`
	GraphVersionID string         `json:"graph_version_id,omitempty"`
}

type NodeUpdateResponse struct {
	NodeID        string `json:"node_id"`
	DomainVersion int    `json:"domain_version"`
	Status        string `json:"status"`
}

type NodeDeleteResponse struct {
	NodeID    string `json:"node_id"`
	IsDeleted bool   `json:"is_deleted"`
}

type NodeDeleteByExternalRefPrefixRequest struct {
	ExternalRefPrefix string `json:"external_ref_prefix"`
}

type NodeDeleteByExternalRefPrefixResponse struct {
	NodeIDs []string `json:"node_ids"`
	Count   int      `json:"count"`
}

type RelationshipBulkDeleteRequest struct {
	RelationshipIDs []string `json:"relationship_ids"`
	GraphVersionID  string   `json:"graph_version_id,omitempty"`
}

type RelationshipBulkDeleteResponse struct {
	RelationshipIDs []string `json:"relationship_ids"`
	Count           int      `json:"count"`
}

type BulkItemError struct {
	Index          int    `json:"index"`
	ExternalRef    string `json:"external_ref,omitempty"`
	RelationshipID string `json:"relationship_id,omitempty"`
	Error          string `json:"error"`
}

type IngestDocumentRequest struct {
	FileURL    string `json:"file_url"`
	LoaiVanBan string `json:"loai_van_ban"`
	DomainID   string `json:"domain_id"`
}

type IngestJobResponse struct {
	JobID        string   `json:"job_id"`
	Status       string   `json:"status"`
	NodesCreated int      `json:"nodes_created,omitempty"`
	Errors       []string `json:"errors,omitempty"`
}

type OpenSyncSessionRequest struct {
	DomainID   string `json:"domain_id"`
	GraphScope string `json:"graph_scope"`
}

type SyncSessionResponse struct {
	SessionID          string `json:"session_id"`
	GraphVersionID     string `json:"graph_version_id"`
	GraphIdentifierID  string `json:"graph_identifier_id"`
	GraphVersionNumber int64  `json:"graph_version_number"`
}
