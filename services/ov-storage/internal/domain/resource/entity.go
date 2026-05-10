// Package resource defines domain entities for OpenViking resource ingestion.
package resource

import (
	"time"

	"github.com/google/uuid"
)

// Resource represents a parsed content resource.
type Resource struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	SourcePath string    `json:"source_path"`
	ParserType ParserType `json:"parser_type"`
	Sections   []Section  `json:"sections"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Section represents a parsed content section.
type Section struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Level   int    `json:"level"` // heading level
}

// ParserType enumerates supported content parsers.
type ParserType string

const (
	ParserMarkdown  ParserType = "markdown"
	ParserPDF       ParserType = "pdf"
	ParserTreeSitter ParserType = "tree_sitter" // Code parsing
)

// WatchEvent represents a file system change notification.
type WatchEvent struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Path      string    `json:"path"`
	EventType string    `json:"event_type"` // created, modified, deleted
	Timestamp time.Time `json:"timestamp"`
}
