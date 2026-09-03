package domain

import "github.com/google/uuid"

// GraphNode represents a node in the knowledge graph.
type GraphNode struct {
	ID         string
	Name       string
	Type       string
	Labels     []string       // [NEW] includes node type + NodeSet tags (e.g. ["Concept", "customer_123", "proj_alpha"])
	Properties map[string]any
	Derived    bool
	VectorID   string
}

// GraphEdge represents a directed edge between two nodes.
type GraphEdge struct {
	ID         string
	SourceID   string
	TargetID   string
	Label      string
	Weight     float64
	Properties map[string]any
	// Memify aliases — used by graph_diff and memify use case
	Subject   string
	Predicate string
	Object    string
	Derived   bool
}

// Entity is a named entity extracted by LLM (used by port.GraphRepository).
type Entity struct {
	ID          string
	Name        string
	Type        string
	Description string
	SourceChunk uuid.UUID // chunk UUID this entity was extracted from
	Properties  map[string]any
}

// NewEntity constructs an Entity with required fields.
// Accepts jobID, name, type, description, sourceChunkID.
func NewEntity(jobID uuid.UUID, name string, entityType EntityType, description string, sourceChunk uuid.UUID) *Entity {
	return &Entity{
		ID:          jobID.String() + "-" + name,
		Name:        name,
		Type:        string(entityType),
		Description: description,
		SourceChunk: sourceChunk,
		Properties:  make(map[string]any),
	}
}

// Community is a cluster of related entities detected by graph algorithms.
type Community struct {
	ID      string
	Summary string
	Members []string // entity IDs in this community
}

// Relationship is an explicit semantic edge between two entities (used by port.GraphRepository).
type Relationship struct {
	ID         string
	SourceID   string
	TargetID   string
	Label      string
	Weight     float64
	Properties map[string]any
}

// PipelineConfig holds configuration for a cognify pipeline run.
type PipelineConfig struct {
	Template PipelineTemplateName // named template
	Steps    []PipelineStep       // custom step list (overrides Template)
	Options  PipelineOptions
}

// Resolve returns the ordered pipeline steps.
// Backward compatible: empty config → STANDARD (7 steps).
func (c PipelineConfig) Resolve() []PipelineStep {
	if c.Template != "" {
		if steps, ok := templateSteps[c.Template]; ok {
			return steps
		}
	}
	if len(c.Steps) > 0 {
		return c.Steps
	}
	return templateSteps[TemplateStandard]
}

// PipelineOptions are per-run tuning parameters.
type PipelineOptions struct {
	ChunkSize    int
	CustomPrompt string
	TemporalMode bool
	SkipCache    bool
}

// NewRelationship constructs a Relationship with full params.
// Accepts: (id uuid.UUID, sourceID, targetID, label string, weight float64, sourceChunk uuid.UUID)
func NewRelationship(id uuid.UUID, sourceID, targetID, label string, weight float64, sourceChunk uuid.UUID) *Relationship {
	return &Relationship{
		ID:       id.String(),
		SourceID: sourceID,
		TargetID: targetID,
		Label:    label,
		Weight:   weight,
		Properties: map[string]any{"source_chunk": sourceChunk.String()},
	}
}

// NewCommunity constructs a Community.
func NewCommunity(id, summary string, members []string) *Community {
	return &Community{ID: id, Summary: summary, Members: members}
}
