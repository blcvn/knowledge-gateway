// Package model defines domain entities for vnp-search-hub.
// Reference: specs/tdd.md §2
package model

import (
	"time"
	"github.com/google/uuid"
)

// RerankStrategy defines the reranking algorithm to use.
type RerankStrategy string

const (
	RerankRRF          RerankStrategy = "rrf"           // Reciprocal Rank Fusion — fast, default
	RerankMMR          RerankStrategy = "mmr"           // Maximal Marginal Relevance — diversity
	RerankCrossEncoder RerankStrategy = "cross_encoder" // Re-score via LLM — highest quality
)

// RecallScope defines which memory types to include.
type RecallScope string

const (
	ScopeAll       RecallScope = "all"
	ScopeSemantic  RecallScope = "semantic"
	ScopeEpisodic  RecallScope = "episodic"
	ScopeProfile   RecallScope = "profile"
	ScopeProcedural RecallScope = "procedural"
	ScopeAdaptive  RecallScope = "adaptive"
	ScopeEvents    RecallScope = "events"
)

// RecallRequest is the input for memory.recall().
type RecallRequest struct {
	Query          string         `json:"query"`
	TenantID       uuid.UUID      `json:"tenant_id"`
	UserID         uuid.UUID      `json:"user_id,omitempty"`
	Scope          RecallScope    `json:"scope"`
	MaxResults     int            `json:"max_results"`
	RerankStrategy RerankStrategy `json:"rerank_strategy"`
	TokenBudget    int            `json:"token_budget"`
}

// RecallResponse is the unified output from memory.recall().
type RecallResponse struct {
	Results      []RecallResult       `json:"results"`
	Context      string               `json:"context"`      // Pre-formatted context window
	Metadata     RecallMetadata       `json:"metadata"`
}

// RecallResult is a single scored result from any engine.
type RecallResult struct {
	Content    string         `json:"content"`
	Score      float64        `json:"score"`
	Source     string         `json:"source"`     // Engine name
	Type       string         `json:"type"`       // fact, event, document, profile, memory
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// RecallMetadata provides observability data.
type RecallMetadata struct {
	LatencyMs    int64    `json:"latency_ms"`
	EnginesUsed  []string `json:"engines_used"`
	TotalResults int      `json:"total_results"`
	TokensUsed   int      `json:"tokens_used"`
}

// EngineResult holds raw results from a single engine.
type EngineResult struct {
	EngineName string         `json:"engine_name"`
	Results    []RecallResult `json:"results"`
	LatencyMs  int64          `json:"latency_ms"`
	Error      error          `json:"-"`
}

// EngineConfig defines connection details for a downstream search engine.
type EngineConfig struct {
	Name    string `json:"name"`
	Address string `json:"address"` // gRPC address
	Enabled bool   `json:"enabled"`
	Timeout time.Duration `json:"timeout"`
}
