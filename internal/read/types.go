package read

import (
	"time"

	"kg-service/internal/platform/graphstore"
	"kg-service/internal/write"
)

const (
	ReadModeNonRealtime = "non-realtime"
	ReadModeRealtime    = "realtime"
)

type TemplateListItem struct {
	TemplateName string            `json:"template_name"`
	Status       string            `json:"status"`
	ParamSchema  []ParamSchemaItem `json:"param_schema"`
}

type ParamSchemaItem struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required,omitempty"`
}

type NodeResponse struct {
	ID            string         `json:"id"`
	NodeType      string         `json:"node_type"`
	DomainID      string         `json:"domain_id"`
	OwnerTenantID string         `json:"owner_tenant_id"`
	OwnerAppID    string         `json:"owner_app_id"`
	Visibility    string         `json:"visibility"`
	SyncVersion   int64          `json:"_kg_sync_version,omitempty"`
	Properties    map[string]any `json:"properties"`
	Relationships []string       `json:"relationships"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// GraphScopeReadRequest asks for every live node and relationship of one graph scope, optionally
// narrowed to specific levels. It is the read counterpart of a scoped write: a client that owns a
// partitioned graph reloads exactly the slice it is about to rewrite.
type GraphScopeReadRequest struct {
	DomainID   string             `json:"domain_id"`
	GraphScope string             `json:"graph_scope"`
	Levels     []write.ScopeLevel `json:"levels,omitempty"`
	// RefsOnly drops property payloads, leaving identity plus partition attributes. A caller
	// computing which rows to delete needs only that, and a full read of a large slice purely to
	// diff identifiers is wasteful.
	RefsOnly bool                 `json:"refs_only,omitempty"`
	Limit    int                  `json:"limit,omitempty"`
	Cursor   GraphScopeReadCursor `json:"cursor,omitempty"`
}

// GraphScopeReadCursor paginates nodes and relationships independently. One combined cursor would
// have to advance both collections in lockstep, which they do not: a scope can hold ten times more
// nodes than edges, or the reverse.
//
// The Done flags exist because a position alone cannot express exhaustion: an empty position means
// "start from the beginning", so a collection that ran out first would silently restart while the
// other was still paging, and the caller would receive its rows twice.
type GraphScopeReadCursor struct {
	Nodes             string `json:"nodes,omitempty"`
	Relationships     string `json:"relationships,omitempty"`
	NodesDone         bool   `json:"nodes_done,omitempty"`
	RelationshipsDone bool   `json:"relationships_done,omitempty"`
}

// IsZero reports whether both collections are exhausted, which is what "no more pages" means.
func (c GraphScopeReadCursor) IsZero() bool {
	return c.NodesDone && c.RelationshipsDone
}

type GraphScopeReadResponse struct {
	Nodes         []GraphScopeNode         `json:"nodes"`
	Relationships []GraphScopeRelationship `json:"relationships"`
	NextCursor    GraphScopeReadCursor     `json:"next_cursor"`
	// HasMore is derived from NextCursor so a client does not have to know the emptiness rule.
	HasMore bool `json:"has_more"`
}

type GraphScopeNode struct {
	ID            string         `json:"id"`
	NodeType      string         `json:"node_type"`
	DomainID      string         `json:"domain_id"`
	OwnerTenantID string         `json:"owner_tenant_id"`
	OwnerAppID    string         `json:"owner_app_id,omitempty"`
	Visibility    string         `json:"visibility"`
	ExternalRef   string         `json:"external_ref,omitempty"`
	Properties    map[string]any `json:"properties"`
	DomainVersion int            `json:"domain_version"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type GraphScopeRelationship struct {
	ID            string         `json:"id"`
	RelType       string         `json:"rel_type"`
	FromNodeID    string         `json:"from_node_id"`
	ToNodeID      string         `json:"to_node_id"`
	DomainID      string         `json:"domain_id"`
	OwnerTenantID string         `json:"owner_tenant_id"`
	OwnerAppID    string         `json:"owner_app_id,omitempty"`
	ExternalRef   string         `json:"external_ref,omitempty"`
	Properties    map[string]any `json:"properties"`
	DomainVersion int            `json:"domain_version"`
	CreatedAt     time.Time      `json:"created_at"`
}

type TemplateExecutionRequest struct {
	Params map[string]any `json:"params"`
	AppID  string         `json:"app_id,omitempty"`
	Mode   string         `json:"mode,omitempty"`
}

type TemplateExecutionResponse struct {
	Results     []map[string]any `json:"results"`
	QueryTimeMs int              `json:"query_time_ms"`
}

type GraphSearchRequest struct {
	AppID         string                  `json:"app_id,omitempty"`
	Mode          string                  `json:"mode,omitempty"`
	DomainID      string                  `json:"domain_id"`
	StartNodeType string                  `json:"start_node_type"`
	StartMatch    map[string]any          `json:"start_match"`
	Hops          []GraphSearchHopRequest `json:"hops"`
	ReturnFields  []string                `json:"return_fields"`
	Params        map[string]any          `json:"params"`
}

type GraphSearchHopRequest struct {
	RelType      string         `json:"rel_type"`
	ToNodeType   string         `json:"to_node_type"`
	Direction    string         `json:"direction"`
	Filter       map[string]any `json:"filter"`
	FilterStatus string         `json:"filter_status"`
}

func (r GraphSearchRequest) ToGraphQuery() graphstore.GraphQuery {
	query := graphstore.GraphQuery{
		StartNodeType: r.StartNodeType,
		StartMatch:    r.StartMatch,
		ReturnFields:  append([]string(nil), r.ReturnFields...),
	}
	for _, hop := range r.Hops {
		query.Hops = append(query.Hops, graphstore.GraphQueryHop{
			RelType:      hop.RelType,
			ToNodeType:   hop.ToNodeType,
			Direction:    hop.Direction,
			Filter:       hop.Filter,
			FilterStatus: hop.FilterStatus,
		})
	}
	return query
}
