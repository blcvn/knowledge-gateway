package write

import (
	"fmt"
	"strings"
	"time"
)

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

	// ExternalRef is the caller-owned identity of this edge. When set, a write against an
	// existing reference updates that relationship (reviving it if soft-deleted) instead of
	// inserting a second one. Omitting it keeps the pre-existing insert-only behaviour.
	ExternalRef string `json:"external_ref,omitempty"`

	// FromExternalRef / ToExternalRef address the endpoints by their nodes' external_ref instead
	// of by kg_node UUID. A client that owns its own identifiers does not know the UUIDs the
	// service minted, and resolving them client-side would mean re-reading every node before
	// every edge write. Ignored when the corresponding *NodeID is set.
	FromExternalRef string `json:"from_external_ref,omitempty"`
	ToExternalRef   string `json:"to_external_ref,omitempty"`
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
	ExternalRef   string         `json:"external_ref,omitempty"`
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

// ScopeLevel names one partition inside a graph scope. A client that keeps its graph split into a
// shared level plus per-feature slices addresses those slices with (level, feature_ref); a client
// that does not partition never sends these at all.
type ScopeLevel struct {
	Level      string `json:"level"`
	FeatureRef string `json:"feature_ref,omitempty"`
}

// ScopeFilter selects rows by graph scope, optionally narrowed to specific levels. An empty Levels
// slice means "every level in this scope" — the whole graph — which is a distinct, meaningful
// request rather than a degenerate one, so it must not be confused with an unset filter.
type ScopeFilter struct {
	DomainID   string       `json:"domain_id"`
	GraphScope string       `json:"graph_scope"`
	Levels     []ScopeLevel `json:"levels,omitempty"`
}

// Matches reports whether a property bag belongs to this filter's levels. Kept next to the filter
// so the in-memory store and the SQL builder cannot drift apart on what "in scope" means.
func (f ScopeFilter) Matches(properties map[string]any) bool {
	if len(f.Levels) == 0 {
		return true
	}
	level := propertyString(properties, "kg_level")
	featureRef := propertyString(properties, "feature_ref")
	for _, want := range f.Levels {
		if want.Level != level {
			continue
		}
		if strings.TrimSpace(want.FeatureRef) == "" || want.FeatureRef == featureRef {
			return true
		}
	}
	return false
}

func propertyString(properties map[string]any, key string) string {
	if properties == nil {
		return ""
	}
	value, ok := properties[key]
	if !ok || value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}

// ScopeQuery is a paginated read over a ScopeFilter.
type ScopeQuery struct {
	ScopeFilter
	// RefsOnly trades content for cost: the caller gets just enough to compute a delete delta
	// (identity plus partition attributes) without transferring every property of every row.
	RefsOnly bool   `json:"refs_only,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Cursor   string `json:"cursor,omitempty"`
}

// ScopeDeleteRequest soft-deletes every node and relationship matching the filter, as part of an
// open sync session. GraphVersionID is required: an unversioned scope-wide delete would leave no
// record of what was removed.
type ScopeDeleteRequest struct {
	ScopeFilter
	GraphVersionID string `json:"graph_version_id"`
}

type ScopeDeleteResponse struct {
	NodeIDs         []string `json:"node_ids"`
	RelationshipIDs []string `json:"relationship_ids"`
	Count           int      `json:"count"`
}

// RelationshipDeleteByExternalRefRequest removes relationships the caller identifies by its own
// references. It is the delta-delete counterpart of upserting by external_ref: a client that owns
// its identifiers never learns the service-side UUIDs.
type RelationshipDeleteByExternalRefRequest struct {
	ExternalRefs   []string `json:"external_refs"`
	GraphVersionID string   `json:"graph_version_id,omitempty"`
}

type RelationshipDeleteByExternalRefResponse struct {
	RelationshipIDs []string `json:"relationship_ids"`
	ExternalRefs    []string `json:"external_refs"`
	Count           int      `json:"count"`
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

// MaxScopePageSize bounds a single scope page. A caller that asks for everything still gets it, one
// page at a time; without a ceiling one request could try to materialise an entire graph.
//
// Exported because every store implementation must clamp identically: if the in-memory store and
// the Postgres store disagreed on the page size or on when a cursor is returned, tests written
// against one would stop proving anything about the other.
const MaxScopePageSize = 1000

// ScopePageLimit clamps a caller-supplied limit into the allowed range.
func ScopePageLimit(limit int) int {
	if limit <= 0 || limit > MaxScopePageSize {
		return MaxScopePageSize
	}
	return limit
}

// ScopeNextCursor returns a cursor only when the page came back full. A short page means the scope
// is exhausted; returning a cursor there would cost the caller an extra empty round trip.
func ScopeNextCursor(pageLen, limit int, lastID string) string {
	if pageLen < limit {
		return ""
	}
	return lastID
}
