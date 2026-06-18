package graphstore_test

import (
	"testing"

	"kg-service/internal/platform/conformance"
	"kg-service/internal/platform/graphstore"
)

func TestInMemoryGraphAdapterConformance(t *testing.T) {
	conformance.AssertGraphAdapterConformance(t, graphstore.NewInMemoryGraphAdapter())
}
