package domain

import "context"

// RequirementType defines the type of a requirement node.
type RequirementType string

const (
	ReqTypeUserStory     RequirementType = "USER_STORY"
	ReqTypeFunctional    RequirementType = "FUNCTIONAL"
	ReqTypeNonFunctional RequirementType = "NON_FUNCTIONAL"
	ReqTypeConstraint    RequirementType = "CONSTRAINT"
	ReqTypeAssumption    RequirementType = "ASSUMPTION"
	ReqTypeUseCase       RequirementType = "USE_CASE"
	ReqTypeFlowStep      RequirementType = "FLOW_STEP"
	ReqTypeAPI           RequirementType = "API"
	ReqTypeEntity        RequirementType = "ENTITY"
	ReqTypePersona       RequirementType = "PERSONA"
	ReqTypeBusinessRule  RequirementType = "BUSINESS_RULE"
)

// DependencyType defines the type of relationship between requirements.
type DependencyType string

const (
	DepTypeRequires  DependencyType = "REQUIRES"   // A requires B
	DepTypeRelatesTo DependencyType = "RELATES_TO" // A relates to B
	DepTypeRefines   DependencyType = "REFINES"    // A refines B (e.g., FR refines Story)
	DepTypeConflicts DependencyType = "CONFLICTS"  // A conflicts with B
)

// RequirementNode represents a single node in the requirement graph.
type RequirementNode struct {
	ID          string          `json:"id"`
	Type        RequirementType `json:"type"`
	Summary     string          `json:"summary"`
	Description string          `json:"description"`
	SourceID    string          `json:"source_id,omitempty"` // ID of the source block/story this node was derived from
	Metadata    map[string]any  `json:"metadata,omitempty"`
}

// DependencyEdge represents a directed edge between two requirement nodes.
type DependencyEdge struct {
	SourceID string         `json:"source_id"`
	TargetID string         `json:"target_id"`
	Type     DependencyType `json:"type"`
	Reason   string         `json:"reason,omitempty"` // Why this dependency exists
}

// RequirementGraph represents the full graph of requirements.
type RequirementGraph struct {
	Nodes    []RequirementNode `json:"nodes"`
	Edges    []DependencyEdge  `json:"edges"`
	Metadata map[string]any    `json:"metadata,omitempty"`
}

// NewRequirementGraph creates an empty graph.
func NewRequirementGraph() *RequirementGraph {
	return &RequirementGraph{
		Nodes: make([]RequirementNode, 0),
		Edges: make([]DependencyEdge, 0),
	}
}

// AddNode adds a node to the graph.
func (g *RequirementGraph) AddNode(node RequirementNode) {
	g.Nodes = append(g.Nodes, node)
}

// GetNode returns a node by ID.
func (g *RequirementGraph) GetNode(id string) (*RequirementNode, bool) {
	for i := range g.Nodes {
		if g.Nodes[i].ID == id {
			return &g.Nodes[i], true
		}
	}
	return nil, false
}

// AddEdge adds an edge to the graph.
func (g *RequirementGraph) AddEdge(edge DependencyEdge) {
	g.Edges = append(g.Edges, edge)
}

// GraphRepository defines the interface for graph persistence
type GraphRepository interface {
	SaveGraph(ctx context.Context, projectID string, graph *RequirementGraph) error
	GetGraph(ctx context.Context, projectID string) (*RequirementGraph, error)
	AddNode(ctx context.Context, projectID string, node RequirementNode) error
	AddEdge(ctx context.Context, projectID string, edge DependencyEdge) error
}
