package vectorstore_test

import (
	"testing"

	"kg-service/internal/platform/conformance"
	"kg-service/internal/platform/vectorstore"
)

func TestInMemoryVectorAdapterConformance(t *testing.T) {
	conformance.AssertVectorAdapterConformance(t, vectorstore.NewInMemoryVectorAdapter())
}
