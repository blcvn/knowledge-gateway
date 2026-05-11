package domain

import (
	"fmt"
	"math"
)

// NodeLabel identifies the type of graph node.
type NodeLabel string

const (
	LabelEntity   NodeLabel = "Entity"
	LabelEpisodic NodeLabel = "Episodic"
	LabelCommunity NodeLabel = "Community"
	LabelSaga     NodeLabel = "Saga"
)

// String implements fmt.Stringer.
func (l NodeLabel) String() string { return string(l) }

// Validate checks if the label is one of the known types.
func (l NodeLabel) Validate() error {
	switch l {
	case LabelEntity, LabelEpisodic, LabelCommunity, LabelSaga:
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrInvalidNodeLabel, l)
	}
}

// EdgeType identifies the type of graph relationship.
type EdgeType string

const (
	EdgeRelatesTo   EdgeType = "RELATES_TO"    // Entity → Entity (bi-temporal fact)
	EdgeMentions    EdgeType = "MENTIONS"       // Episodic → Entity
	EdgeHasMember   EdgeType = "HAS_MEMBER"    // Community → Entity
	EdgeHasEpisode  EdgeType = "HAS_EPISODE"   // Saga → Episodic
	EdgeNextEpisode EdgeType = "NEXT_EPISODE"  // Episodic → Episodic (temporal order)
)

// String implements fmt.Stringer.
func (t EdgeType) String() string { return string(t) }

// GroupID represents a tenant partition identifier.
// All queries are scoped by GroupID for multi-tenant isolation.
type GroupID string

// String implements fmt.Stringer.
func (g GroupID) String() string { return string(g) }

// Validate ensures the GroupID is non-empty.
func (g GroupID) Validate() error {
	if g == "" {
		return ErrMissingGroupID
	}
	return nil
}

// EmbeddingVector represents a dense float32 vector for similarity search.
type EmbeddingVector []float32

// Dimension returns the number of elements in the vector.
func (v EmbeddingVector) Dimension() int {
	return len(v)
}

// Validate checks that the vector has the expected dimension.
func (v EmbeddingVector) Validate(expectedDim int) error {
	if len(v) == 0 {
		return ErrEmptyEmbedding
	}
	if len(v) != expectedDim {
		return fmt.Errorf("%w: got %d, expected %d", ErrInvalidEmbeddingDimension, len(v), expectedDim)
	}
	return nil
}

// CosineSimilarity computes cosine similarity between two vectors.
// Returns a value between -1 and 1, where 1 = identical direction.
func (v EmbeddingVector) CosineSimilarity(other EmbeddingVector) (float64, error) {
	if len(v) != len(other) {
		return 0, fmt.Errorf("%w: %d vs %d", ErrInvalidEmbeddingDimension, len(v), len(other))
	}
	if len(v) == 0 {
		return 0, ErrEmptyEmbedding
	}

	var dotProduct, normA, normB float64
	for i := range v {
		a, b := float64(v[i]), float64(other[i])
		dotProduct += a * b
		normA += a * a
		normB += b * b
	}

	denominator := math.Sqrt(normA) * math.Sqrt(normB)
	if denominator == 0 {
		return 0, nil
	}
	return dotProduct / denominator, nil
}

// SimilarityMetric identifies the distance function for vector search.
type SimilarityMetric string

const (
	MetricCosine    SimilarityMetric = "cosine"
	MetricEuclidean SimilarityMetric = "euclidean"
)

// PaginationOpts controls cursor-based pagination for list queries.
type PaginationOpts struct {
	Cursor string `json:"cursor,omitempty"` // Last seen UUID for cursor pagination
	Limit  int    `json:"limit"`            // Max results to return
}

// DefaultPagination returns PaginationOpts with sensible defaults.
func DefaultPagination() PaginationOpts {
	return PaginationOpts{Limit: 50}
}
