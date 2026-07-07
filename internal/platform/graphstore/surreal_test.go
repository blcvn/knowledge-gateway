package graphstore_test

import (
	"strings"
	"testing"

	"kg-service/internal/platform/conformance"
	"kg-service/internal/platform/graphstore"
)

func TestBuildSurrealTraversal_NoHops(t *testing.T) {
	got := graphstore.ExportBuildSurrealTraversal(nil)
	if got != "" {
		t.Fatalf("want empty string, got %q", got)
	}
}

func TestBuildSurrealTraversal_OneHopOut(t *testing.T) {
	hops := []graphstore.GraphQueryHop{
		{RelType: "LINKS", ToNodeType: "Document", Direction: "out"},
	}
	got := graphstore.ExportBuildSurrealTraversal(hops)
	if !strings.Contains(got, "->kg_rel_LINKS->kg_node_Document") {
		t.Fatalf("unexpected traversal: %q", got)
	}
}

func TestBuildSurrealTraversal_OneHopIn(t *testing.T) {
	hops := []graphstore.GraphQueryHop{
		{RelType: "LINKS", ToNodeType: "Document", Direction: "in"},
	}
	got := graphstore.ExportBuildSurrealTraversal(hops)
	if !strings.Contains(got, "<-kg_rel_LINKS<-kg_node_Document") {
		t.Fatalf("unexpected traversal: %q", got)
	}
}

func TestBuildSurrealTraversal_TwoHops(t *testing.T) {
	hops := []graphstore.GraphQueryHop{
		{RelType: "OWNS", ToNodeType: "Org", Direction: "out"},
		{RelType: "MEMBER", ToNodeType: "User", Direction: "in"},
	}
	got := graphstore.ExportBuildSurrealTraversal(hops)
	if !strings.Contains(got, "->kg_rel_OWNS->kg_node_Org") {
		t.Fatalf("missing first hop: %q", got)
	}
	if !strings.Contains(got, "<-kg_rel_MEMBER<-kg_node_User") {
		t.Fatalf("missing second hop: %q", got)
	}
}

func TestSurrealGraphAdapter_FallbackWithNoEndpoint(t *testing.T) {
	adapter := graphstore.NewSurrealGraphAdapter(graphstore.SurrealConfig{})
	if adapter == nil {
		t.Fatal("NewSurrealGraphAdapter returned nil")
	}
	// No endpoint → delegates to in-memory, should not panic or error.
	conformance.AssertGraphAdapterConformance(t, adapter)
}
