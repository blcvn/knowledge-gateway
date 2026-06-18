package graphstore

import (
	"context"
	"time"
)

type GraphAdapter interface {
	UpsertNode(ctx context.Context, node GraphNode) error
	DeleteNode(ctx context.Context, nodeID string) error
	UpsertRelationship(ctx context.Context, rel GraphRelationship) error
	DeleteRelationship(ctx context.Context, relID string) error
	ExecuteQuery(ctx context.Context, query GraphQuery, params map[string]any) ([]map[string]any, error)
	ListNodes(ctx context.Context) ([]GraphNode, error)
	ListRelationships(ctx context.Context) ([]GraphRelationship, error)
	ReadSyncVersion(ctx context.Context, entityID string) (int64, error)
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

type GraphQuery struct {
	StartNodeType  string
	StartMatch     map[string]any
	Hops           []GraphQueryHop
	ReturnFields   []string
	ACLTokensParam string
	MaxDepth       int
	Strategy       string
}

type GraphQueryHop struct {
	RelType      string
	ToNodeType   string
	Direction    string
	Filter       map[string]any
	FilterStatus string
}
