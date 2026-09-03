// Package graphiti defines domain entities for the Knowledge Graph.
//
// Absorbed from: graphiti-ingestion, graphiti-knowledge, graphiti-store, graphiti-search, graphiti-pipeline
// (MERGE-P2-T1)
package graphiti

import "time"

// Episode is a raw input event ingested into the knowledge graph.
type Episode struct {
	UUID      string     `json:"uuid"`
	Name      string     `json:"name"`
	Content   string     `json:"content"`
	Source    string     `json:"source"`    // "message" | "document" | "json"
	SourceID  string     `json:"source_id"` // Optional external reference ID
	TenantID  string     `json:"tenant_id"`
	Embedding []float32  `json:"embedding,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// Node represents a knowledge graph entity (Person, Place, Concept, etc.)
type Node struct {
	UUID       string         `json:"uuid"`
	Name       string         `json:"name"`
	Type       string         `json:"type"` // Entity type label
	Summary    string         `json:"summary"`
	Attributes map[string]any `json:"attributes,omitempty"`
	TenantID   string         `json:"tenant_id"`
	Episodes   []string       `json:"episodes,omitempty"` // Related episode UUIDs
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// Edge represents a directed relation between two graph nodes.
type Edge struct {
	UUID       string    `json:"uuid"`
	SourceUUID string    `json:"source_uuid"`
	TargetUUID string    `json:"target_uuid"`
	Relation   string    `json:"relation"` // Relation label (e.g. KNOWS, WORKS_AT)
	Weight     float64   `json:"weight"`
	Facts      []Fact    `json:"facts,omitempty"`
	TenantID   string    `json:"tenant_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// Fact is a temporal statement associated with an edge.
type Fact struct {
	Content   string     `json:"content"`
	Valid     bool       `json:"valid"`
	ValidAt   *time.Time `json:"valid_at,omitempty"`
	InvalidAt *time.Time `json:"invalid_at,omitempty"`
}

// Ontology describes the schema of entity types and relationships in a tenant's graph.
type Ontology struct {
	TenantID    string         `json:"tenant_id"`
	EntityTypes []string       `json:"entity_types"`
	Relations   []RelationType `json:"relations"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// RelationType describes a relation in the ontology.
type RelationType struct {
	Name        string `json:"name"`
	SourceType  string `json:"source_type"`
	TargetType  string `json:"target_type"`
	Description string `json:"description,omitempty"`
}

// SearchQuery is a knowledge graph search request.
type SearchQuery struct {
	Query    string         `json:"query"`
	TenantID string         `json:"tenant_id"`
	Mode     string         `json:"mode"`  // "semantic" | "graph" | "hybrid"
	Limit    int            `json:"limit"`
	Filter   map[string]any `json:"filter,omitempty"`
}

// SearchResult aggregates results from graph search.
type SearchResult struct {
	Episodes []*Episode `json:"episodes,omitempty"`
	Nodes    []*Node    `json:"nodes,omitempty"`
	Edges    []*Edge    `json:"edges,omitempty"`
	Score    float64    `json:"score"`
}

// IngestRequest carries input for episode ingestion.
type IngestRequest struct {
	Name     string `json:"name"`
	Content  string `json:"content"`
	Source   string `json:"source"`
	SourceID string `json:"source_id,omitempty"`
	TenantID string `json:"tenant_id"`
}
