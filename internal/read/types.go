package read

import (
	"time"

	"kg-service/internal/platform/graphstore"
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
