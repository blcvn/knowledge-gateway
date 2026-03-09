package biz

type FullGraphResult struct {
	Nodes      []NodeResult
	Edges      []EdgeResult
	TotalNodes int
	TotalEdges int
}

type NodeResult struct {
	ID         string
	Labels     []string
	Properties map[string]any
}

type EdgeResult struct {
	ID           string
	RelationType string
	SourceNodeID string
	TargetNodeID string
	Properties   map[string]any
}
