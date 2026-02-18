package v32

// Note: This file is a direct port of graph.go from the reference implementation
// adapted for the v3.2 domain package in ba-agent-service.

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
	ReqTypeGlossary      RequirementType = "GLOSSARY"
	ReqTypeBusinessRule  RequirementType = "BUSINESS_RULE"
	ReqTypeAnalytics     RequirementType = "ANALYTICS"
	ReqTypeMetric        RequirementType = "METRIC"
	ReqTypeUserFlow      RequirementType = "USER_FLOW"
	ReqTypePermission    RequirementType = "PERMISSION"
	ReqTypeScope         RequirementType = "SCOPE"
	ReqTypeChangeRecord  RequirementType = "CHANGE_RECORD"
	ReqTypeHistory       RequirementType = "HISTORY"
)

// DependencyType defines the type of relationship between requirements.
type DependencyType string

const (
	DepTypeRequires  DependencyType = "REQUIRES"   // A requires B
	DepTypeRelatesTo DependencyType = "RELATES_TO" // A relates to B
	DepTypeRefines   DependencyType = "REFINES"    // A refines B (e.g., FR refines Story)
	DepTypeConflicts DependencyType = "CONFLICTS"  // A conflicts with B
	DepTypeAdded     DependencyType = "ADDED"
	DepTypeModified  DependencyType = "MODIFIED"
	DepTypeDeleted   DependencyType = "DELETED"
)

// RequirementNode represents a single node in the requirement graph.
type RequirementNode struct {
	ID          string          `json:"id" gorm:"primaryKey"`
	DocumentID  string          `json:"document_id" gorm:"index"`  // Link to Document
	ReferenceID string          `json:"reference_id" gorm:"index"` // Original Semantic ID (e.g., US-001)
	Type        RequirementType `json:"type"`
	Summary     string          `json:"summary"`
	Description string          `json:"description" gorm:"type:text"`
	SourceID    string          `json:"source_id,omitempty"` // ID of the source block/story this node was derived from
	Metadata    map[string]any  `json:"metadata,omitempty" gorm:"serializer:json"`
}

// DependencyEdge represents a directed edge between two requirement nodes.
type DependencyEdge struct {
	ID         string         `json:"id" gorm:"primaryKey"` // Added ID for GORM
	DocumentID string         `json:"document_id" gorm:"index"`
	SourceID   string         `json:"source_id" gorm:"index"`
	TargetID   string         `json:"target_id" gorm:"index"`
	Type       DependencyType `json:"type"`
	Reason     string         `json:"reason,omitempty" gorm:"type:text"` // Why this dependency exists
}

// RequirementGraph represents the full graph of requirements.
type RequirementGraph struct {
	ID         string            `json:"id" gorm:"primaryKey"`
	DocumentID string            `json:"document_id" gorm:"index"`
	Nodes      []RequirementNode `json:"nodes" gorm:"-"` // Handled manually in Repo
	Edges      []DependencyEdge  `json:"edges" gorm:"-"` // Handled manually in Repo
	Metadata   map[string]any    `json:"metadata,omitempty" gorm:"serializer:json"`
}

// NewRequirementGraph creates an empty graph.
func NewRequirementGraph() *RequirementGraph {
	return &RequirementGraph{
		Nodes:    make([]RequirementNode, 0),
		Edges:    make([]DependencyEdge, 0),
		Metadata: make(map[string]any),
	}
}

// AddNode adds a node to the graph. It avoids adding duplicate nodes with the same ID.
func (g *RequirementGraph) AddNode(node RequirementNode) {
	for _, n := range g.Nodes {
		if n.ID == node.ID {
			// Update existing node if needed? For now, skip if exists as per reference
			// But reference actually returns if exists.
			// However, in kg_builder, we often want to UPDATE.
			// The reference AddNode implementation simply returns if ID matches.
			return
		}
	}
	g.Nodes = append(g.Nodes, node)
}

// AddEdge adds an edge to the graph. It avoids adding duplicate edges with the same ID.
func (g *RequirementGraph) AddEdge(edge DependencyEdge) {
	for _, e := range g.Edges {
		if e.ID == edge.ID {
			return
		}
	}
	g.Edges = append(g.Edges, edge)
}

// GetNode retrieves a node by ID. Returns nil and false if not found.
func (g *RequirementGraph) GetNode(id string) (*RequirementNode, bool) {
	for i := range g.Nodes {
		if g.Nodes[i].ID == id {
			return &g.Nodes[i], true
		}
	}
	return nil, false
}
