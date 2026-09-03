package data

import (
	"strings"
	"testing"
)

func TestBuildGetFullGraphNodesQuery(t *testing.T) {
	query := buildGetFullGraphNodesQuery()
	required := []string{
		"MATCH (n:Entity {app_id: $app_id, tenant_id: $tenant_id})",
		"RETURN n",
		"ORDER BY n.id",
		"SKIP $offset",
		"LIMIT $limit",
	}
	for _, token := range required {
		if !strings.Contains(query, token) {
			t.Fatalf("query missing token %q\nquery:\n%s", token, query)
		}
	}
}

func TestBuildGetFullGraphEdgesQuery(t *testing.T) {
	query := buildGetFullGraphEdgesQuery()
	required := []string{
		"MATCH (a:Entity {app_id: $app_id, tenant_id: $tenant_id})-[r]->(b:Entity {app_id: $app_id, tenant_id: $tenant_id})",
		"WHERE a.id IN $node_ids AND b.id IN $node_ids",
		"RETURN r, type(r) AS rel_type, a.id AS source_id, b.id AS target_id",
		"ORDER BY r.id",
	}
	for _, token := range required {
		if !strings.Contains(query, token) {
			t.Fatalf("query missing token %q\nquery:\n%s", token, query)
		}
	}
}

func TestBuildCountQueries(t *testing.T) {
	nodeCountQuery := buildCountNodesQuery()
	edgeCountQuery := buildCountEdgesQuery()
	if !strings.Contains(nodeCountQuery, "RETURN count(n) AS total") {
		t.Fatalf("node count query is invalid: %s", nodeCountQuery)
	}
	if !strings.Contains(edgeCountQuery, "RETURN count(r) AS total") {
		t.Fatalf("edge count query is invalid: %s", edgeCountQuery)
	}
}
