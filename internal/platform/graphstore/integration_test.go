package graphstore_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"kg-service/internal/platform/graphstore"
)

func TestNeo4jGraphAdapterRealBackendIntegration(t *testing.T) {
	endpoint := requiredEnv(t, "KG_NEO4J_ENDPOINT", "KG_GRAPH_ENDPOINT")
	runRealGraphBackendIntegration(t, graphstore.NewNeo4jGraphAdapter(graphstore.CypherConfig{
		Endpoint: endpoint,
		Database: os.Getenv("KG_GRAPH_DATABASE"),
	}))
}

func TestMemgraphGraphAdapterRealBackendIntegration(t *testing.T) {
	endpoint := requiredEnv(t, "KG_MEMGRAPH_ENDPOINT", "KG_GRAPH_ENDPOINT")
	runRealGraphBackendIntegration(t, graphstore.NewMemgraphGraphAdapter(graphstore.CypherConfig{
		Endpoint: endpoint,
		Database: os.Getenv("KG_GRAPH_DATABASE"),
	}))
}

func TestMemgraphGraphAdapterBatchBackendIntegration(t *testing.T) {
	endpoint := requiredEnv(t, "KG_MEMGRAPH_ENDPOINT", "KG_GRAPH_ENDPOINT")
	adapter := graphstore.NewMemgraphGraphAdapter(graphstore.CypherConfig{
		Endpoint: endpoint,
		Database: os.Getenv("KG_GRAPH_DATABASE"),
	})

	ctx := context.Background()
	suffix := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_"))
	suffix = strings.ReplaceAll(suffix, "-", "_")
	suffix = fmt.Sprintf("%s_%d", suffix, time.Now().UnixNano())

	nodeA := graphstore.GraphNode{
		ID:            suffix + "_a",
		NodeType:      "Doc",
		DomainID:      "integration-domain",
		OwnerTenantID: "tenant",
		OwnerAppID:    "app",
		ACLVisibleTo:  []string{"tenant:app"},
		SyncVersion:   21,
		Properties: map[string]any{
			"title": suffix + "-alpha",
		},
	}
	nodeB := graphstore.GraphNode{
		ID:            suffix + "_b",
		NodeType:      "Doc",
		DomainID:      "integration-domain",
		OwnerTenantID: "tenant",
		OwnerAppID:    "app",
		ACLVisibleTo:  []string{"tenant:app"},
		SyncVersion:   22,
		Properties: map[string]any{
			"title": suffix + "-beta",
		},
	}
	rel := graphstore.GraphRelationship{
		ID:          suffix + "_rel",
		RelType:     "LINKS",
		FromNodeID:  nodeA.ID,
		ToNodeID:    nodeB.ID,
		DomainID:    "integration-domain",
		SyncVersion: 23,
		Properties: map[string]any{
			"kind": "integration-batch",
		},
	}

	must := func(err error) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	must(adapter.UpsertNodesBatch(ctx, []graphstore.GraphNode{nodeA, nodeB}))
	must(adapter.UpsertRelationshipsBatch(ctx, []graphstore.GraphRelationship{rel}))

	t.Cleanup(func() {
		_ = adapter.DeleteRelationshipsBatch(ctx, []graphstore.GraphRelationship{rel})
		_ = adapter.DeleteNodesBatch(ctx, []graphstore.GraphNode{nodeB, nodeA})
	})

	nodes, err := adapter.ListNodes(ctx)
	if err != nil {
		t.Fatalf("ListNodes() error = %v", err)
	}
	if !hasNode(nodes, nodeA.ID) || !hasNode(nodes, nodeB.ID) {
		t.Fatalf("ListNodes() missing batch integration nodes: %#v", nodes)
	}

	rels, err := adapter.ListRelationships(ctx)
	if err != nil {
		t.Fatalf("ListRelationships() error = %v", err)
	}
	if !hasRelationship(rels, rel.ID) {
		t.Fatalf("ListRelationships() missing batch integration relationship: %#v", rels)
	}
}

func requiredEnv(t *testing.T, keys ...string) string {
	t.Helper()
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	t.Skipf("skipping integration test; set one of %s", strings.Join(keys, ", "))
	return ""
}

func runRealGraphBackendIntegration(t *testing.T, adapter graphstore.GraphAdapter) {
	t.Helper()

	ctx := context.Background()
	suffix := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_"))
	suffix = strings.ReplaceAll(suffix, "-", "_")
	suffix = fmt.Sprintf("%s_%d", suffix, time.Now().UnixNano())

	nodeA := graphstore.GraphNode{
		ID:            suffix + "_a",
		NodeType:      "Doc",
		DomainID:      "integration-domain",
		OwnerTenantID: "tenant",
		OwnerAppID:    "app",
		ACLVisibleTo:  []string{"tenant:app"},
		SyncVersion:   11,
		Properties: map[string]any{
			"title": suffix + "-alpha",
		},
	}
	nodeB := graphstore.GraphNode{
		ID:            suffix + "_b",
		NodeType:      "Doc",
		DomainID:      "integration-domain",
		OwnerTenantID: "tenant",
		OwnerAppID:    "app",
		ACLVisibleTo:  []string{"tenant:app"},
		SyncVersion:   12,
		Properties: map[string]any{
			"title": suffix + "-beta",
		},
	}
	rel := graphstore.GraphRelationship{
		ID:          suffix + "_rel",
		RelType:     "LINKS",
		FromNodeID:  nodeA.ID,
		ToNodeID:    nodeB.ID,
		DomainID:    "integration-domain",
		SyncVersion: 13,
		Properties: map[string]any{
			"kind": "integration",
		},
	}

	must := func(err error) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	must(adapter.UpsertNode(ctx, nodeA))
	must(adapter.UpsertNode(ctx, nodeB))
	must(adapter.UpsertRelationship(ctx, rel))

	t.Cleanup(func() {
		_ = adapter.DeleteRelationship(ctx, rel.ID)
		_ = adapter.DeleteNode(ctx, nodeB.ID)
		_ = adapter.DeleteNode(ctx, nodeA.ID)
	})

	nodes, err := adapter.ListNodes(ctx)
	if err != nil {
		t.Fatalf("ListNodes() error = %v", err)
	}
	if !hasNode(nodes, nodeA.ID) || !hasNode(nodes, nodeB.ID) {
		t.Fatalf("ListNodes() missing integration nodes: %#v", nodes)
	}

	rels, err := adapter.ListRelationships(ctx)
	if err != nil {
		t.Fatalf("ListRelationships() error = %v", err)
	}
	if !hasRelationship(rels, rel.ID) {
		t.Fatalf("ListRelationships() missing integration relationship: %#v", rels)
	}

	version, err := adapter.ReadSyncVersion(ctx, nodeA.ID)
	if err != nil {
		t.Fatalf("ReadSyncVersion() error = %v", err)
	}
	if version != nodeA.SyncVersion {
		t.Fatalf("ReadSyncVersion() = %d, want %d", version, nodeA.SyncVersion)
	}
}

func hasNode(nodes []graphstore.GraphNode, id string) bool {
	for _, node := range nodes {
		if node.ID == id {
			return true
		}
	}
	return false
}

func hasRelationship(rels []graphstore.GraphRelationship, id string) bool {
	for _, rel := range rels {
		if rel.ID == id {
			return true
		}
	}
	return false
}
