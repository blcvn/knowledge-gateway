package memory

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type RoleType string

const (
	RoleTypeNoRole    RoleType = "norole"
	RoleTypeSystem    RoleType = "system"
	RoleTypeAssistant RoleType = "assistant"
	RoleTypeUser      RoleType = "user"
	RoleTypeFunction  RoleType = "function"
	RoleTypeTool      RoleType = "tool"
)

// Message represents an atomic interaction in a thread.
type Message struct {
	UUID       uuid.UUID
	ThreadUUID uuid.UUID
	Role       string
	RoleType   RoleType
	Content    string
	TokenCount int
	Metadata   map[string]interface{}
	CreatedAt  time.Time
	DeletedAt  *time.Time
}

// MemoryContext represents the enriched context assembled for LLMs.
type MemoryContext struct {
	Messages       []*Message               `json:"messages"`
	RelevantFacts  []map[string]interface{} `json:"facts"`
	SystemPrompt   string                   `json:"system_prompt,omitempty"`
}

// Repository defines the contract for persisting messages.
type Repository interface {
	Save(ctx context.Context, msg *Message) error
	GetLastN(ctx context.Context, threadUUID uuid.UUID, n int) ([]*Message, error)
}
