// Package vectorstore defines a unified interface for vector similarity search
// backends. VNP Memory supports both Qdrant and pgvector as dual backends.
//
// Each engine selects its backend via configuration. The interface abstracts
// the storage layer so engines can switch backends without code changes.
//
// See: specs/technical/TECH-001-migrate-qdrant-to-pgvector.md
package vectorstore

import "context"

// Vector is a float32 embedding vector.
type Vector = []float32

// Document represents a vector with associated metadata.
type Document struct {
	// ID uniquely identifies this document within a collection.
	ID string `json:"id"`

	// Vector is the embedding representation.
	Vector Vector `json:"vector"`

	// Metadata holds arbitrary key-value pairs for filtering.
	Metadata map[string]any `json:"metadata,omitempty"`

	// Content is the original text content (optional, for retrieval).
	Content string `json:"content,omitempty"`

	// Score is populated on search results (cosine similarity).
	Score float64 `json:"score,omitempty"`
}

// SearchParams configures a similarity search query.
type SearchParams struct {
	// Vector is the query embedding.
	Vector Vector

	// TopK is the maximum number of results to return.
	TopK int

	// MinScore filters results below this similarity threshold (0.0–1.0).
	MinScore float64

	// Filter restricts results to documents matching these metadata conditions.
	Filter map[string]any

	// Collection targets a specific collection/table.
	Collection string
}

// CollectionConfig defines a vector collection's schema.
type CollectionConfig struct {
	// Name is the collection identifier.
	Name string

	// Dimension is the vector dimensionality (e.g., 1536 for OpenAI ada-002).
	Dimension int

	// DistanceMetric defines the similarity function.
	DistanceMetric DistanceMetric
}

// DistanceMetric defines supported similarity functions.
type DistanceMetric string

const (
	Cosine    DistanceMetric = "cosine"
	Euclidean DistanceMetric = "euclidean"
	DotProduct DistanceMetric = "dot_product"
)

// VectorStore is the unified interface for vector similarity search backends.
// Both Qdrant and pgvector implement this interface.
type VectorStore interface {
	// EnsureCollection creates a collection if it doesn't exist.
	EnsureCollection(ctx context.Context, cfg CollectionConfig) error

	// Upsert inserts or updates documents in a collection.
	Upsert(ctx context.Context, collection string, docs []Document) error

	// Search performs similarity search and returns top-K results.
	Search(ctx context.Context, params SearchParams) ([]Document, error)

	// Delete removes documents by IDs from a collection.
	Delete(ctx context.Context, collection string, ids []string) error

	// DropCollection removes a collection entirely.
	DropCollection(ctx context.Context, collection string) error

	// Close releases backend connections.
	Close() error
}
