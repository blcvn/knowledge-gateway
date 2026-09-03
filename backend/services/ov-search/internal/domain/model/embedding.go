package model

import "time"

type EmbeddingVector struct {
	ID           string    `json:"id"`
	Vector       []float32 `json:"vector"`        // 1536-dim
	SparseVector []float32 `json:"sparse_vector"` // BM25 sparse
}

type UpsertPayload struct {
	Path         string    `json:"path"`
	AccountID    string    `json:"account_id"`
	UserID       string    `json:"user_id"`
	ContentHash  string    `json:"content_hash"`
	ContextLevel string    `json:"context_level"`
	ChunkIndex   int       `json:"chunk_index"`
	ParentDir    string    `json:"parent_dir"`
	MimeType     string    `json:"mime_type"`
	UpdatedAt    time.Time `json:"updated_at"`
}
