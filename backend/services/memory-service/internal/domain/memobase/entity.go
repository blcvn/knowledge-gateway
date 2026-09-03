// Package memobase defines domain entities for working memory.
//
// Absorbed from: memobase-context, memobase-engine, memobase-ingestion, memobase-pipeline
// (MERGE-P2-T3)
package memobase

import "time"

// Blob is an atomic unit of memory content.
type Blob struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	TenantID  string         `json:"tenant_id"`
	Type      string         `json:"type"` // "conversation"|"fact"|"document"|"image"
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Embedding []float32      `json:"embedding,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// UserContext is the aggregated memory context for a user.
type UserContext struct {
	UserID   string     `json:"user_id"`
	TenantID string     `json:"tenant_id"`
	Summary  string     `json:"summary"`
	Profiles []*Profile `json:"profiles,omitempty"`
	Events   []*Event   `json:"events,omitempty"`
	Tokens   int        `json:"tokens"`
}

// Profile is a structured user attribute extracted from memory.
type Profile struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Category  string    `json:"category"` // "preference"|"fact"|"goal"|"habit"
	Score     float64   `json:"score"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Event is a timestamped activity entry.
type Event struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	EventType string         `json:"event_type"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// Buffer is the pending blob buffer for a user (before flush).
type Buffer struct {
	UserID         string  `json:"user_id"`
	Blobs          []*Blob `json:"blobs"`
	TokenCount     int     `json:"token_count"`
	FlushThreshold int     `json:"flush_threshold"`
}
