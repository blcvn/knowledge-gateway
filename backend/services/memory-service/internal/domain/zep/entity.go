// Package zep defines domain entities for Zep memory service integration.
//
// Absorbed from: zep-user, zep-thread, zep-memory, zep-search, zep-graph, zep-admin
// (MERGE-P2-T3)
package zep

import "time"

// ZepUser represents a user in the Zep memory system.
type ZepUser struct {
	UserID    string         `json:"user_id"`
	Email     string         `json:"email"`
	FirstName string         `json:"first_name"`
	LastName  string         `json:"last_name"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// ZepSession represents a conversation session.
type ZepSession struct {
	SessionID string         `json:"session_id"`
	UserID    string         `json:"user_id"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// ZepMemory aggregates messages, summary, and facts for a session.
type ZepMemory struct {
	SessionID string       `json:"session_id"`
	Messages  []ZepMessage `json:"messages,omitempty"`
	Summary   *ZepSummary  `json:"summary,omitempty"`
	Facts     []string     `json:"facts,omitempty"`
}

// ZepMessage is a single message in a session.
type ZepMessage struct {
	Role      string         `json:"role"` // "user"|"assistant"
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// ZepSummary is an AI-generated session summary.
type ZepSummary struct {
	Content  string    `json:"content"`
	TokensUsed int     `json:"tokens_used"`
	CreatedAt time.Time `json:"created_at"`
}

// GraphFact is a knowledge graph fact associated with a user.
type GraphFact struct {
	UUID     string   `json:"uuid"`
	Name     string   `json:"name"`
	Fact     string   `json:"fact"`
	Episodes []string `json:"episodes,omitempty"`
}
