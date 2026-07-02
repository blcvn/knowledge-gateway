package bridge

import "time"

type RawNode struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	Name           string `json:"name"`
	QualifiedName  string `json:"qualified_name"`
	FilePath       string `json:"file_path"`
	Language       string `json:"language"`
	StartLine      int    `json:"start_line"`
	EndLine        int    `json:"end_line"`
	StartColumn    int    `json:"start_column"`
	EndColumn      int    `json:"end_column"`
	Docstring      string `json:"docstring"`
	Signature      string `json:"signature"`
	Visibility     string `json:"visibility"`
	IsExported     int    `json:"is_exported"`
	IsAsync        int    `json:"is_async"`
	IsStatic       int    `json:"is_static"`
	IsAbstract     int    `json:"is_abstract"`
	Decorators     string `json:"decorators"`
	TypeParameters string `json:"type_parameters"`
	ReturnType     string `json:"return_type"`
}

type RawEdge struct {
	Source     string `json:"source"`
	Target     string `json:"target"`
	Kind       string `json:"kind"`
	Metadata   string `json:"metadata"`
	Line       int    `json:"line"`
	Col        int    `json:"col"`
	Provenance string `json:"provenance"`
}

type NodeSpec struct {
	ExternalRef string
	NodeType    string
	Visibility  string
	Properties  map[string]any
	SourceID    string
	SourceKind  string
}

type EdgeSpec struct {
	Key              string
	RelType          string
	FromExternalRef  string
	ToExternalRef    string
	Properties       map[string]any
	SourceKind       string
	SourceProvenance string
}

type Graph struct {
	Nodes []NodeSpec
	Edges []EdgeSpec
}

type SyncReport struct {
	ProjectID            string
	CommitSHA            string
	NodeCount            int
	RelationshipCount    int
	CreatedNodes         int
	UpdatedNodes         int
	DeletedNodes         int
	CreatedRelationships int
	DeletedRelationships int
	SkippedRelationships int
	Duration             time.Duration
	OutputDir            string
	NodeTypeCounts       map[string]int
	EdgeTypeCounts       map[string]int
}

type State struct {
	Nodes         map[string]StateNode         `json:"nodes"`
	Relationships map[string]StateRelationship `json:"relationships"`
}

type StateNode struct {
	ID       string `json:"id"`
	NodeType string `json:"node_type"`
}

type StateRelationship struct {
	ID      string `json:"id"`
	RelType string `json:"rel_type"`
	FromID  string `json:"from_id"`
	ToID    string `json:"to_id"`
}
