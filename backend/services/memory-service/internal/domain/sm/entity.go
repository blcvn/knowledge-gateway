// Package sm defines domain entities for Supermemory integration.
//
// Absorbed from: sm-memory, sm-document, sm-profile
// (MERGE-P2-T3)
package sm

import "time"

// SMMemory is a Supermemory entry.
type SMMemory struct {
	ID        string         `json:"id"`
	TenantID  string         `json:"tenant_id"`
	Content   string         `json:"content"`
	Tags      []string       `json:"tags,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Embedding []float32      `json:"embedding,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// SMDocument is a managed document.
type SMDocument struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Type      string    `json:"type"` // "markdown"|"pdf"|"html"
	URL       string    `json:"url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// SMProfile aggregates user memories and statistics.
type SMProfile struct {
	UserID   string      `json:"user_id"`
	TenantID string      `json:"tenant_id"`
	Memories []*SMMemory `json:"memories,omitempty"`
	Tags     []string    `json:"tags,omitempty"`
	Stats    ProfileStats `json:"stats"`
}

// ProfileStats contains aggregate user statistics.
type ProfileStats struct {
	TotalMemories int   `json:"total_memories"`
	TotalTokens   int64 `json:"total_tokens"`
}

// RAGResponse is the result of a RAG (Retrieval-Augmented Generation) query.
type RAGResponse struct {
	Context string      `json:"context"`
	Sources []*SMMemory `json:"sources,omitempty"`
	Tokens  int         `json:"tokens"`
}
