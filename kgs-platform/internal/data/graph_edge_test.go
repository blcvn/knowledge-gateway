package data

import (
	"strings"
	"testing"
)

func TestBuildCreateEdgeQueryUsesMerge(t *testing.T) {
	query := buildCreateEdgeQuery("RELATES_TO")
	required := []string{
		"MATCH (a {app_id: $app_id, tenant_id: $tenant_id, id: $source_node_id})",
		"MATCH (b {app_id: $app_id, tenant_id: $tenant_id, id: $target_node_id})",
		"MERGE (a)-[rel:RELATES_TO {app_id: $app_id, tenant_id: $tenant_id, id: $edge_id}]->(b)",
		"ON CREATE SET rel += $props, rel.created_at = datetime()",
		"ON MATCH SET rel += $props, rel.updated_at = datetime()",
	}
	for _, token := range required {
		if !strings.Contains(query, token) {
			t.Fatalf("query missing token %q\nquery:\n%s", token, query)
		}
	}
	if strings.Contains(query, "CREATE (a)-[rel:") {
		t.Fatalf("query should not use CREATE for relationship upsert")
	}
}
