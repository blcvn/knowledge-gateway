//go:build e2e

package e2e

import "testing"

func TestErrorPaths_EnvironmentGuardrails(t *testing.T) {
	env := setupE2EEnv(t)
	defer env.teardown(t)

	if env.endpoints["neo4j"] == env.endpoints["redis"] {
		t.Fatalf("invalid environment: duplicated dependency endpoint")
	}
}
