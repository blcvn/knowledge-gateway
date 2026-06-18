//go:build nebula

package graphstore_test

import (
	"os"
	"testing"

	"kg-service/internal/platform/graphstore"
)

func TestNebulaGraphAdapterRealBackendIntegration(t *testing.T) {
	endpoint := requiredEnv(t, "KG_NEBULA_ENDPOINT", "KG_GRAPH_ENDPOINT")
	runRealGraphBackendIntegration(t, graphstore.NewNebulaGraphAdapter(graphstore.CypherConfig{
		Endpoint: endpoint,
		Database: os.Getenv("KG_GRAPH_DATABASE"),
	}))
}
