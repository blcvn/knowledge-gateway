package graph

import "time"

// OntologyProperty defines a typed property within an entity/edge schema
type OntologyProperty struct {
	Name        string `json:"name"`
	Type        string `json:"type"`        // string | number | boolean | datetime
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// EntityTypeSchema — prescribed entity type definition.
// When registered for a group_id, LLM extraction is constrained to these types.
type EntityTypeSchema struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Properties  []OntologyProperty `json:"properties,omitempty"`
	Examples    []string           `json:"examples,omitempty"`
}

// EdgeTypeSchema — prescribed relationship type definition.
// Constrains LLM to only extract relationships of these types between valid source/target types.
type EdgeTypeSchema struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	SourceTypes []string           `json:"source_types,omitempty"` // allowed source entity types
	TargetTypes []string           `json:"target_types,omitempty"` // allowed target entity types
	Properties  []OntologyProperty `json:"properties,omitempty"`
	Examples    []string           `json:"examples,omitempty"`
}

// OntologyRegistry — per group_id ontology configuration.
// nil/empty = "learned ontology" (LLM chooses any label freely).
// Non-empty = "prescribed ontology" (LLM constrained to defined types).
type OntologyRegistry struct {
	GroupID     string                      `json:"group_id"`
	EntityTypes map[string]EntityTypeSchema `json:"entity_types"` // key: type name
	EdgeTypes   map[string]EdgeTypeSchema   `json:"edge_types"`   // key: relation name
	CreatedAt   time.Time                   `json:"created_at"`
	UpdatedAt   time.Time                   `json:"updated_at"`
}

// IsPrescribed returns true if this registry has defined types (not learned mode)
func (r *OntologyRegistry) IsPrescribed() bool {
	return r != nil && len(r.EntityTypes) > 0
}
