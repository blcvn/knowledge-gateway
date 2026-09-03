package port

import (
	"time"

	"vnp-memory/services/graphiti-store/domain"
)

// --- Request DTOs ---

// SaveNodeRequest contains fields for creating/updating an entity node.
type SaveNodeRequest struct {
	UUID          string            `json:"uuid"`
	Name          string            `json:"name"`
	GroupID       string            `json:"group_id"`
	Summary       string            `json:"summary,omitempty"`
	NameEmbedding []float32         `json:"name_embedding,omitempty"`
	Labels        []string          `json:"labels,omitempty"`
	Attributes    map[string]string `json:"attributes,omitempty"`
}

// SaveCommunityRequest contains fields for creating/updating a community node.
type SaveCommunityRequest struct {
	UUID          string            `json:"uuid"`
	Name          string            `json:"name"`
	GroupID       string            `json:"group_id"`
	Summary       string            `json:"summary,omitempty"`
	NameEmbedding []float32         `json:"name_embedding,omitempty"`
	Level         int               `json:"level"`
}

// SaveEdgeRequest contains fields for creating a bi-temporal relationship.
type SaveEdgeRequest struct {
	UUID          string            `json:"uuid"`
	SourceNodeID  string            `json:"source_node_id"`
	TargetNodeID  string            `json:"target_node_id"`
	Name          string            `json:"name"`
	GroupID       string            `json:"group_id"`
	Fact          string            `json:"fact"`
	FactEmbedding []float32         `json:"fact_embedding,omitempty"`
	ValidAt       time.Time         `json:"valid_at"`
	InvalidAt     *time.Time        `json:"invalid_at,omitempty"`
	Attributes    map[string]string `json:"attributes,omitempty"`
	EpisodeID     string            `json:"episode_id"`
}

// InvalidateEdgeRequest marks an edge as no longer valid.
type InvalidateEdgeRequest struct {
	UUID      string    `json:"uuid"`
	InvalidAt time.Time `json:"invalid_at"`
}

// VectorSearchRequest for cosine similarity search.
type VectorSearchRequest struct {
	Embedding domain.EmbeddingVector `json:"embedding"`
	GroupID   string                 `json:"group_id"`
	Limit     int                    `json:"limit"`
}

// TextSearchRequest for BM25 fulltext search.
type TextSearchRequest struct {
	Query   string `json:"query"`
	GroupID string `json:"group_id"`
	Limit   int    `json:"limit"`
}

// BFSSearchRequest for graph traversal.
type BFSSearchRequest struct {
	StartNodeID string `json:"start_node_id"`
	MaxDepth    int    `json:"max_depth"`
	GroupID     string `json:"group_id"`
	Limit       int    `json:"limit"`
}

// SaveBulkRequest for atomic batch persistence.
type SaveBulkRequest struct {
	Nodes   []domain.EntityNode  `json:"nodes"`
	Edges   []domain.EntityEdge  `json:"edges"`
	Episode domain.EpisodicNode  `json:"episode"`
}

// --- Response DTOs ---

// SaveBulkResponse returns counts of persisted items.
type SaveBulkResponse struct {
	NodeCount int    `json:"node_count"`
	EdgeCount int    `json:"edge_count"`
	EpisodeID string `json:"episode_id"`
}
