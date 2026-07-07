//go:build !nebula

package graphstore_test

import (
	"testing"

	"kg-service/internal/platform/conformance"
	"kg-service/internal/platform/graphstore"
)

func TestInMemoryGraphAdapterConformance(t *testing.T) {
	conformance.AssertGraphAdapterConformance(t, graphstore.NewInMemoryGraphAdapter())
}

func TestNeo4jGraphAdapterConformance(t *testing.T) {
	conformance.AssertGraphAdapterConformance(t, graphstore.NewNeo4jGraphAdapter(graphstore.CypherConfig{}))
}

func TestMemgraphGraphAdapterConformance(t *testing.T) {
	conformance.AssertGraphAdapterConformance(t, graphstore.NewMemgraphGraphAdapter(graphstore.CypherConfig{}))
}

func TestNebulaGraphAdapterConformance(t *testing.T) {
	conformance.AssertGraphAdapterConformance(t, graphstore.NewNebulaGraphAdapter(graphstore.CypherConfig{}))
}

// TestSurrealGraphAdapterConformance runs against in-memory fallback when no
// endpoint is configured (passes always). Set SURREAL_ENDPOINT env to test
// against a live SurrealDB instance.
func TestSurrealGraphAdapterConformance(t *testing.T) {
	conformance.AssertGraphAdapterConformance(t, graphstore.NewSurrealGraphAdapter(graphstore.SurrealConfig{}))
}

