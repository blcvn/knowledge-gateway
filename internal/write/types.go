package write

import "time"

type NodeCreateRequest struct {
	DomainID      string         `json:"domain_id"`
	NodeType      string         `json:"node_type"`
	Properties    map[string]any `json:"properties"`
	Visibility    string         `json:"visibility"`
	ExternalRef   string         `json:"external_ref"`
	BridgeNodeIDs []string       `json:"bridge_node_ids,omitempty"`
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
}

type RelationshipCreateRequest struct {
	RelType    string         `json:"rel_type"`
	FromNodeID string         `json:"from_node_id"`
	ToNodeID   string         `json:"to_node_id"`
	DomainID   string         `json:"domain_id"`
	Properties map[string]any `json:"properties"`
}

type RelationshipRecord struct {
	ID            string         `json:"id"`
	RelType       string         `json:"rel_type"`
	FromNodeID    string         `json:"from_node_id"`
	ToNodeID      string         `json:"to_node_id"`
	DomainID      string         `json:"domain_id"`
	OwnerTenantID string         `json:"owner_tenant_id"`
	OwnerAppID    string         `json:"owner_app_id"`
	Properties    map[string]any `json:"properties"`
	CreatedAt     time.Time      `json:"created_at"`
}

type RelationshipCreateResponse struct {
	RelationshipID string `json:"relationship_id"`
	Status         string `json:"status"`
}

type NodeCreateResponse struct {
	NodeID        string `json:"node_id"`
	DomainVersion int    `json:"domain_version"`
	Status        string `json:"status"`
	SyncETAMs     int    `json:"sync_eta_ms"`
}

type NodeUpdateRequest struct {
	Properties  map[string]any `json:"properties"`
	Visibility  string         `json:"visibility"`
	ExternalRef string         `json:"external_ref"`
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
