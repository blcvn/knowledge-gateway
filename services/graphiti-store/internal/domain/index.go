package domain

// IndexType identifies the type of database index.
type IndexType string

const (
	IndexVector    IndexType = "vector"     // Vector similarity (cosine/euclidean)
	IndexFulltext  IndexType = "fulltext"   // BM25 text search
	IndexComposite IndexType = "composite"  // Multi-property index
	IndexRange     IndexType = "range"      // Single-property range index
)

// String implements fmt.Stringer.
func (t IndexType) String() string { return string(t) }

// IndexDefinition describes a graph database index.
type IndexDefinition struct {
	Name             string           `json:"name"`
	Type             IndexType        `json:"type"`
	TargetLabel      string           `json:"target_label"`       // Node label or relationship type
	Properties       []string         `json:"properties"`         // Properties to index
	VectorDimension  int              `json:"vector_dimension,omitempty"`  // For vector indexes
	SimilarityFunc   SimilarityMetric `json:"similarity_func,omitempty"`   // For vector indexes
}

// DefaultIndexes returns the standard set of indexes for a Graphiti graph.
func DefaultIndexes(vectorDim int) []IndexDefinition {
	return []IndexDefinition{
		{
			Name:            "entity_name_embedding",
			Type:            IndexVector,
			TargetLabel:     "Entity",
			Properties:      []string{"name_embedding"},
			VectorDimension: vectorDim,
			SimilarityFunc:  MetricCosine,
		},
		{
			Name:            "edge_fact_embedding",
			Type:            IndexVector,
			TargetLabel:     "RELATES_TO",
			Properties:      []string{"fact_embedding"},
			VectorDimension: vectorDim,
			SimilarityFunc:  MetricCosine,
		},
		{
			Name:        "entity_name_fulltext",
			Type:        IndexFulltext,
			TargetLabel: "Entity",
			Properties:  []string{"name", "summary"},
		},
		{
			Name:        "edge_fact_fulltext",
			Type:        IndexFulltext,
			TargetLabel: "RELATES_TO",
			Properties:  []string{"name", "fact"},
		},
		{
			Name:        "entity_group_id",
			Type:        IndexRange,
			TargetLabel: "Entity",
			Properties:  []string{"group_id"},
		},
		{
			Name:        "edge_temporal",
			Type:        IndexComposite,
			TargetLabel: "RELATES_TO",
			Properties:  []string{"group_id", "valid_at", "invalid_at"},
		},
	}
}
