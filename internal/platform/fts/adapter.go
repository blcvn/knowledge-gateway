package fts

import (
	"context"
	"time"
)

type FTSAdapter interface {
	Index(ctx context.Context, doc FTSDocument) error
	Delete(ctx context.Context, nodeID string) error
	Search(ctx context.Context, query FTSQuery, filter FTSFilter, opts SearchOptions) ([]FTSResult, error)
}

type FTSDocument struct {
	NodeID         string            `json:"node_id"`
	NodeType       string            `json:"node_type"`
	DomainID       string            `json:"domain_id"`
	OwnerTenantID  string            `json:"owner_tenant_id"`
	OwnerAppID     string            `json:"owner_app_id"`
	ACLVisibleTo   []string          `json:"acl_visible_to"`
	IsDeleted      bool              `json:"is_deleted"`
	StatusValue    string            `json:"status_value,omitempty"`
	AuthorityScore *int              `json:"authority_score,omitempty"`
	DomainProps    map[string]any    `json:"domain_props"`
	Fields         map[string]string `json:"fields,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
}

type FTSQuery struct {
	Text   string
	Mode   string
	Fields []string
}

type FTSFilter struct {
	DomainIDs      []string
	ACLVisibleTo   []string
	NodeTypes      []string
	OwnerTenantIDs []string
	OwnerAppIDs    []string
}

type FTSResult struct {
	Document FTSDocument
	Score    float64
}

type SearchOptions struct {
	TopK int
}
