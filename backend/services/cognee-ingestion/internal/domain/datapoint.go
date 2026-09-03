// Package domain defines the DataPoint entity for structured ingestion without LLM.
// TASK-CE-007: DataPoint Schema (SOL-003 - Structured Ingestion, Zero LLM)
//
// DataPoints are the opposite of unstructured ingestion:
//   - User provides explicit schema (type, fields, index_fields, relations)
//   - System maps directly to Neo4j nodes + Qdrant embeddings
//   - NO LLM extraction step — zero-latency ingestion
//   - Same ID submitted twice → upsert (version incremented)
package domain

import (
	"crypto/sha1"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DataPoint — atomic knowledge unit with stable UUID identity.
// Designed for structured data (employees, papers, products, etc.)
// where the schema is known in advance and LLM extraction is unnecessary.
type DataPoint struct {
	ID          uuid.UUID           // Stable: deterministic UUID from content hash (idempotent)
	Version     int                 // Increment on update, start at 1
	DatasetID   uuid.UUID
	TenantID    string
	Type        string              // Schema type: "Paper", "User", "Product", "Employee"
	Fields      map[string]any      // All field values (dynamic schema, stored as JSONB)
	IndexFields []string            // Only these fields embedded into Qdrant (cost control)
	Relations   []DataPointRelation // Explicit FK edges to other DataPoints
	NodeSets    []string            // NodeSet tags (CR-002 integration for memory scoping)
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DataPointRelation — explicit FK edge to another DataPoint.
// Maps to a Neo4j relationship: (source)-[:Label]->(target)
type DataPointRelation struct {
	TargetID uuid.UUID
	Label    string  // Edge label: "authored_by", "belongs_to", "cites", "works_in"
	Weight   float64 // Default 1.0
}

// DeterministicUUID generates a stable UUID from namespace + key.
// Same content → same UUID → idempotent ingestion (upsert semantics).
func DeterministicUUID(namespace, key string) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(namespace+":"+key))
}

// ContentHash generates a SHA1 hash of type + fields for deterministic ID generation.
func ContentHash(typeName string, fields map[string]any) string {
	h := sha1.New()
	fmt.Fprintf(h, "%s:%v", typeName, fields)
	return fmt.Sprintf("%x", h.Sum(nil))
}
