package search

import "time"

type SemanticSearchRequest struct {
	Query     string   `json:"query"`
	DomainIDs []string `json:"domain_ids"`
	TopK      int      `json:"top_k"`
}

type SemanticSearchResponse struct {
	Results      []SearchResult `json:"results"`
	SearchTimeMs int            `json:"search_time_ms"`
}

type FullTextSearchRequest struct {
	Query     string   `json:"query"`
	DomainIDs []string `json:"domain_ids"`
	TopK      int      `json:"top_k"`
	Mode      string   `json:"mode"`
	Fields    []string `json:"fields"`
}

type FullTextSearchResponse struct {
	Results      []SearchResult `json:"results"`
	SearchTimeMs int            `json:"search_time_ms"`
}

type HybridSearchRequest struct {
	Query          string   `json:"query"`
	DomainIDs      []string `json:"domain_ids"`
	TopK           int      `json:"top_k"`
	FTSOperator    string   `json:"fts_operator"`
	SemanticWeight float64  `json:"semantic_weight"`
}

type HybridSearchResponse struct {
	Results      []SearchResult `json:"results"`
	SearchTimeMs int            `json:"search_time_ms"`
}

type SearchResult struct {
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
	Content        string         `json:"content"`
	Score          float64        `json:"score"`
	CreatedAt      time.Time      `json:"created_at"`
}
