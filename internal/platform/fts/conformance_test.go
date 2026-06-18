package fts_test

import (
	"testing"

	"kg-service/internal/platform/conformance"
	"kg-service/internal/platform/fts"
)

func TestInMemoryFTSAdapterConformance(t *testing.T) {
	conformance.AssertFTSAdapterConformance(t, fts.NewInMemoryFTSAdapter())
}
