package vectorstore

import (
	"strings"
	"testing"
)

func TestVectorLiteral(t *testing.T) {
	if got := vectorLiteral([]float64{1, 2.5, 3}); got != "[1,2.5,3]" {
		t.Fatalf("vectorLiteral() = %q, want [1,2.5,3]", got)
	}
}

func TestPgVectorBuildANNStatementPreFilter(t *testing.T) {
	adapter := NewPgVectorAdapter(nil)
	stmt, args := adapter.buildANNStatement([]float64{1, 0}, VectorFilter{
		DomainIDs:      []string{"d1"},
		ACLVisibleTo:   []string{"t:a"},
		OwnerTenantIDs: []string{"t"},
	}, ANNOptions{
		TopK:       20,
		MinScore:   0.2,
		IndexHint:  "hnsw",
		EfSearch:   77,
		FilterMode: "pre",
	})

	wantContains := []string{
		"SET LOCAL enable_seqscan = off",
		"SET LOCAL hnsw.ef_search = 77",
		"FROM kg_vector_documents",
		"domain_id = ANY($2)",
		"owner_tenant_id = ANY($3)",
		"acl_visible_to && $4::text[]",
		"LIMIT $5",
	}
	for _, want := range wantContains {
		if !strings.Contains(stmt, want) {
			t.Fatalf("statement missing %q: %s", want, stmt)
		}
	}
	if len(args) != 5 {
		t.Fatalf("args len = %d, want 5", len(args))
	}
	if args[0] != "[1,0]" {
		t.Fatalf("args[0] = %v, want vector literal", args[0])
	}
}

func TestPgVectorBuildANNStatementPostFilter(t *testing.T) {
	adapter := NewPgVectorAdapter(nil)
	stmt, args := adapter.buildANNStatement([]float64{1, 0}, VectorFilter{
		DomainIDs: []string{"d1"},
	}, ANNOptions{
		TopK:       5,
		FilterMode: "post",
	})

	if !strings.Contains(stmt, "WITH ranked AS (") {
		t.Fatalf("post-filter statement missing ranked CTE: %s", stmt)
	}
	if !strings.Contains(stmt, "WHERE domain_id = ANY($2)") {
		t.Fatalf("post-filter statement missing outer filter: %s", stmt)
	}
	if !strings.Contains(stmt, "LIMIT $3") {
		t.Fatalf("post-filter statement missing limit: %s", stmt)
	}
	if len(args) != 3 {
		t.Fatalf("args len = %d, want 3", len(args))
	}
}
