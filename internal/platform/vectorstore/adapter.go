package vectorstore

import "context"

type VectorAdapter interface {
	Upsert(ctx context.Context, doc VectorDocument) error
	Delete(ctx context.Context, nodeID string) error
	ANN(ctx context.Context, query []float64, filter VectorFilter, opts ANNOptions) ([]VectorResult, error)
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

type VectorFilter struct {
	DomainIDs      []string
	ACLVisibleTo   []string
	NodeTypes      []string
	OwnerTenantIDs []string
	OwnerAppIDs    []string
}

type VectorResult struct {
	Document VectorDocument
	Score    float64
}

type ANNOptions struct {
	TopK       int
	MinScore   float64
	IndexHint  string
	EfSearch   int
	FilterMode string
}
