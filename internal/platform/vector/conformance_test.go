package vector_test

import (
	"testing"

	"kg-service/internal/platform/conformance"
	"kg-service/internal/platform/vector"
)

func TestDeterministicProviderConformance(t *testing.T) {
	conformance.AssertEmbeddingProviderConformance(t, vector.NewDeterministicProvider(8))
}
