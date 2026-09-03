package data

import (
	"strings"
	"testing"
)

func TestBuildDeleteNodeEdgeCountQuery(t *testing.T) {
	query := buildDeleteNodeEdgeCountQuery()
	required := []string{
		"MATCH (n {app_id: $app_id, tenant_id: $tenant_id, id: $node_id})",
		"OPTIONAL MATCH (n)-[r]-()",
		"RETURN count(r) AS edge_count",
	}
	for _, token := range required {
		if !strings.Contains(query, token) {
			t.Fatalf("delete-node count query missing %q\nquery:\n%s", token, query)
		}
	}
}

func TestBuildDeleteNodeQuery(t *testing.T) {
	query := buildDeleteNodeQuery()
	required := []string{
		"MATCH (n {app_id: $app_id, tenant_id: $tenant_id, id: $node_id})",
		"DETACH DELETE n",
	}
	for _, token := range required {
		if !strings.Contains(query, token) {
			t.Fatalf("delete-node query missing %q\nquery:\n%s", token, query)
		}
	}
}

func TestBuildDeleteEdgeQuery(t *testing.T) {
	query := buildDeleteEdgeQuery()
	required := []string{
		"MATCH ()-[r {app_id: $app_id, tenant_id: $tenant_id, id: $edge_id}]->()",
		"DELETE r",
	}
	for _, token := range required {
		if !strings.Contains(query, token) {
			t.Fatalf("delete-edge query missing %q\nquery:\n%s", token, query)
		}
	}
}

func TestBuildBatchDeleteNodesQueries(t *testing.T) {
	countQuery := buildBatchDeleteNodesCountQuery()
	countRequired := []string{
		"UNWIND $node_ids AS node_id",
		"OPTIONAL MATCH (n {app_id: $app_id, tenant_id: $tenant_id, id: node_id})",
		"RETURN size(nodes) AS deleted, count(DISTINCT r) AS edges_removed",
	}
	for _, token := range countRequired {
		if !strings.Contains(countQuery, token) {
			t.Fatalf("batch-delete count query missing %q\nquery:\n%s", token, countQuery)
		}
	}

	deleteQuery := buildBatchDeleteNodesQuery()
	deleteRequired := []string{
		"UNWIND $node_ids AS node_id",
		"MATCH (n {app_id: $app_id, tenant_id: $tenant_id, id: node_id})",
		"DETACH DELETE n",
	}
	for _, token := range deleteRequired {
		if !strings.Contains(deleteQuery, token) {
			t.Fatalf("batch-delete query missing %q\nquery:\n%s", token, deleteQuery)
		}
	}
}
