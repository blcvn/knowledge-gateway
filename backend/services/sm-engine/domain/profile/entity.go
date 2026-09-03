// Package profile defines domain entities for Supermemory dynamic user profiles.
package profile

import (
	"time"

	"github.com/google/uuid"
)

// Profile represents a user's adaptive knowledge profile.
type Profile struct {
	ID              uuid.UUID          `json:"id"`
	TenantID        uuid.UUID          `json:"tenant_id"`
	UserID          uuid.UUID          `json:"user_id"`
	StaticPreferences []StaticPreference `json:"static_preferences"` // User-defined
	DynamicTraits   []DynamicTrait     `json:"dynamic_traits"`     // System-inferred
	UpdatedAt       time.Time          `json:"updated_at"`
}

// StaticPreference is a user-declared preference.
type StaticPreference struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// DynamicTrait is a system-inferred behavioral pattern.
type DynamicTrait struct {
	Category   TraitCategory `json:"category"`
	Name       string        `json:"name"`
	Confidence float64       `json:"confidence"` // 0.0 - 1.0
	InferredAt time.Time     `json:"inferred_at"`
}

// TraitCategory classifies dynamic traits.
type TraitCategory string

const (
	TraitInterest   TraitCategory = "interest"
	TraitBehavior   TraitCategory = "behavior"
	TraitPreference TraitCategory = "preference"
	TraitExpertise  TraitCategory = "expertise"
)
