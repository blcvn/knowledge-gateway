//go:build e2e

package e2e

import "testing"

func TestHappyPath_FullStackBootstrap(t *testing.T) {
	env := setupE2EEnv(t)
	defer env.teardown(t)

	for _, dep := range []string{"postgres", "neo4j", "redis", "qdrant", "opa"} {
		if env.endpoints[dep] == "" {
			t.Fatalf("missing endpoint for %s", dep)
		}
	}
}
