// Package domain defines Chunk and related types for text processing.
package domain

import "github.com/google/uuid"

// Chunk is a unit of text produced by the chunking stage.
type Chunk struct {
	ID         uuid.UUID
	Index      int
	Content    string
	CharCount  int
	Source     string
	DatasetID  string
	TenantID   string
	Metadata   map[string]any
}

// NewChunk creates a Chunk with the given parameters.
// Overloaded signature accepts: (jobUUID, index, text, charCount, source)
func NewChunk(id uuid.UUID, index int, content string, charCount int, source string) *Chunk {
	return &Chunk{
		ID:        id,
		Index:     index,
		Content:   content,
		CharCount: charCount,
		Source:    source,
		Metadata:  make(map[string]any),
	}
}
