// Package search defines domain entities for search-service.
//
// Absorbed from: vnp-search-hub, ov-search, sm-search, sm-connector, sm-mcp
// (MERGE-P2-T4)
package search

import "time"

// Query is a unified cross-engine search request.
type Query struct {
	Query        string         `json:"query"`
	TenantID     string         `json:"tenant_id"`
	Engines      []string       `json:"engines,omitempty"` // ["graphiti","cognee","memobase","storage","sm"]
	Mode         string         `json:"mode"`              // "semantic"|"keyword"|"hybrid"
	RankStrategy string         `json:"rank_strategy"`     // "rrf"|"mmr"|"simple"
	Limit        int            `json:"limit"`
	Offset       int            `json:"offset"`
	Filter       map[string]any `json:"filter,omitempty"`
}

// Result is the aggregated search response.
type Result struct {
	Items   []*Item        `json:"items"`
	Total   int            `json:"total"`
	Took    time.Duration  `json:"took_ms"`
	Engines []string       `json:"engines"`
	Facets  map[string]int `json:"facets,omitempty"`
}

// Item is a single search result from any engine.
type Item struct {
	ID        string         `json:"id"`
	Content   string         `json:"content"`
	Score     float64        `json:"score"`
	Source    string         `json:"source"` // "graphiti"|"cognee"|"memobase"|"storage"|"sm"
	Type      string         `json:"type"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Highlight string         `json:"highlight,omitempty"`
}

// RAGResponse is the result of a retrieval-augmented generation query.
type RAGResponse struct {
	Context string  `json:"context"`
	Sources []*Item `json:"sources"`
	Tokens  int     `json:"tokens"`
}

// RRFConfig holds Reciprocal Rank Fusion configuration.
type RRFConfig struct {
	K int `json:"k"` // RRF constant (default: 60)
}
