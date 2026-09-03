package dto

import "vnp-memory/services/cognee-search/internal/domain"

type SearchRequest struct {
	Query      string
	Strategies []string
	TopK       int
	Rerank     bool
	Filters    domain.SearchFilters
}

type RAGRequest struct {
	Query      string
	Strategies []string
	TopK       int
	Filters    domain.SearchFilters
}

type ExploreRequest struct {
	NodeID   string
	Depth    int
	TenantID string
}
