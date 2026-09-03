// Package memory defines domain entities for the Supermemory adaptive memory.
package memory

import (
	"math"
	"time"

	"github.com/google/uuid"
)

// Memory represents a knowledge memory with forgetting curve.
type Memory struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	UserID    uuid.UUID `json:"user_id"`
	Content   string    `json:"content"`
	Category  string    `json:"category"`
	Strength  float64   `json:"strength"`  // 0.0 - 1.0, decays over time
	HalfLife  float64   `json:"half_life"` // Hours until 50% strength
	LastAccess time.Time `json:"last_access"`
	CreatedAt time.Time `json:"created_at"`
}

// CurrentStrength calculates memory strength using Ebbinghaus decay.
// strength = initial_strength × e^(-t / half_life × ln(2))
func (m *Memory) CurrentStrength() float64 {
	elapsed := time.Since(m.LastAccess).Hours()
	decay := math.Exp(-elapsed / m.HalfLife * math.Ln2)
	return m.Strength * decay
}

// Relation represents a connection between memories.
type Relation struct {
	ID         uuid.UUID `json:"id"`
	SourceID   uuid.UUID `json:"source_id"`
	TargetID   uuid.UUID `json:"target_id"`
	Type       string    `json:"type"` // related, contradicts, supersedes
	Weight     float64   `json:"weight"`
	CreatedAt  time.Time `json:"created_at"`
}
