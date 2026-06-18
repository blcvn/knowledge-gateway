package graphstore

import (
	"context"
	"testing"
)

func TestInMemoryGraphAdapterExecuteQuery(t *testing.T) {
	adapter := NewInMemoryGraphAdapter()
	_ = adapter.UpsertNode(context.Background(), GraphNode{
		ID:            "a",
		NodeType:      "Doc",
		DomainID:      "d",
		OwnerTenantID: "t",
		OwnerAppID:    "a",
		ACLVisibleTo:  []string{"t:a"},
		Properties:    map[string]any{"title": "alpha"},
	})
	_ = adapter.UpsertNode(context.Background(), GraphNode{
		ID:            "b",
		NodeType:      "Doc",
		DomainID:      "d",
		OwnerTenantID: "t",
		OwnerAppID:    "a",
		ACLVisibleTo:  []string{"t:a"},
		Properties:    map[string]any{"title": "beta"},
	})
	_ = adapter.UpsertRelationship(context.Background(), GraphRelationship{
		ID:         "r1",
		RelType:    "LINKS",
		FromNodeID: "a",
		ToNodeID:   "b",
		DomainID:   "d",
	})

	results, err := adapter.ExecuteQuery(context.Background(), GraphQuery{
		StartNodeType:  "Doc",
		StartMatch:     map[string]any{"title": "alpha"},
		Hops:           []GraphQueryHop{{RelType: "LINKS", ToNodeType: "Doc"}},
		ReturnFields:   []string{"id", "hop_1"},
		ACLTokensParam: "acl_tokens",
	}, map[string]any{"acl_tokens": []string{"t:a"}})
	if err != nil {
		t.Fatalf("ExecuteQuery() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results len = %d, want 1", len(results))
	}
	if results[0]["hop_1"] != "b" {
		t.Fatalf("hop_1 = %v, want b", results[0]["hop_1"])
	}
}
