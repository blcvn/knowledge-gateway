package dto

import "vnp-memory/services/cognee-search/internal/domain"

type SearchResponse struct {
	Results []domain.SearchResult
}

type RAGResponse struct {
	Answer  string
	Sources []domain.SearchResult
}

type ExploreResponse struct {
	Results []domain.SearchResult
}
