package biz

import "testing"

func TestQueryPlannerLabelFilter(t *testing.T) {
	qp := NewQueryPlanner()

	contextQuery := qp.BuildContextQuery("Requirement", "BOTH")
	if want := ":Requirement"; !contains(contextQuery, want) {
		t.Fatalf("expected context query to contain %q, got %s", want, contextQuery)
	}

	impactQuery := qp.BuildImpactQuery("UseCase", 3)
	if want := "(m:UseCase"; !contains(impactQuery, want) {
		t.Fatalf("expected impact query to contain %q, got %s", want, impactQuery)
	}

	coverageQuery := qp.BuildCoverageQuery("Actor", 2)
	if want := "(m:Actor"; !contains(coverageQuery, want) {
		t.Fatalf("expected coverage query to contain %q, got %s", want, coverageQuery)
	}
}

func TestQueryPlannerBatchedTraversal(t *testing.T) {
	qp := NewQueryPlanner()
	queries := qp.BuildBatchedTraversalQueries("context", "Requirement", "OUTGOING", 5, 3)
	if len(queries) != 2 {
		t.Fatalf("expected 2 batched queries for depth=5, got %d", len(queries))
	}
	if !contains(queries[0], "[*1..3]") || !contains(queries[1], "[*4..5]") {
		t.Fatalf("unexpected depth windows: %#v", queries)
	}
}

func TestPageTokenEncodeDecode(t *testing.T) {
	token := EncodePageToken(123)
	offset, err := DecodePageToken(token)
	if err != nil {
		t.Fatalf("DecodePageToken error: %v", err)
	}
	if offset != 123 {
		t.Fatalf("expected 123, got %d", offset)
	}
}

func contains(s, needle string) bool {
	return len(needle) == 0 || (len(s) >= len(needle) && indexOf(s, needle) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
