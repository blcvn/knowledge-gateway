package dto

import "vnp-memory/ov-search/internal/domain/model"

type SearchRequest struct {
	Query          string
	AccountID      string
	MaxResults     int
	ContextLevel   string // L0, L1, L2
	EnableHotness  bool
	RerankStrategy string // rrf, mmr, cross_encoder
}

type SearchResponse struct {
	Results []model.SearchResult `json:"results"`
}

type ContextRequest struct {
	Path         string
	ContextLevel string
}

type ContextResponse struct {
	Content string
}

type UpsertRequest struct {
	Path        string
	AccountID   string
	Content     string
	ContextLevel string
	UserID      string
}

type DeleteRequest struct {
	Path      string
	AccountID string
}
