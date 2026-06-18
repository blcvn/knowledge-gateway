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
