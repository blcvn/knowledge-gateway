package graphstore

import (
	"context"
	"strings"
	"testing"
)

type recordingCypherRunner struct {
	cypher string
	params map[string]any
	result []map[string]any
}

func (r *recordingCypherRunner) Run(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error) {
	r.cypher = cypher
	r.params = params
	return append([]map[string]any(nil), r.result...), nil
}

func TestGraphQueryToCypherDefaultStrategy(t *testing.T) {
	cypher, params := graphQueryToCypher(GraphQuery{
		StartNodeType:  "Doc",
		StartMatch:     map[string]any{"title": "alpha"},
		ReturnFields:   []string{"id"},
		ACLTokensParam: "acl_tokens",
	}, map[string]any{"acl_tokens": []string{"tenant:app"}})

	if !strings.Contains(cypher, "MATCH (n0:Doc)") {
		t.Fatalf("cypher = %q, want start node", cypher)
	}
	if !strings.Contains(cypher, "n0.title = $n0_title") {
		t.Fatalf("cypher = %q, want start match binding", cypher)
	}
	if !strings.Contains(cypher, "RETURN id") {
		t.Fatalf("cypher = %q, want return clause", cypher)
	}
	if got := params["n0_title"]; got != "alpha" {
		t.Fatalf("params[n0_title] = %#v, want alpha", got)
	}
}

func TestGraphQueryToCypherDeepTraversal(t *testing.T) {
	cypher, _ := graphQueryToCypher(GraphQuery{
		StartNodeType: "Doc",
		Hops: []GraphQueryHop{
			{RelType: "LINKS", ToNodeType: "Doc", Direction: "out"},
		},
		ReturnFields: []string{"id"},
		Strategy:     "deep_traversal",
	}, nil)

	if !strings.Contains(cypher, "/* max_depth:10 strategy:deep_traversal */") {
		t.Fatalf("cypher = %q, want deep traversal marker", cypher)
	}
	if !strings.Contains(cypher, "MATCH (n0)-[r1:LINKS]->(n1:Doc)") {
		t.Fatalf("cypher = %q, want hop match", cypher)
	}
}

func TestGraphQueryToCypherACLBinding(t *testing.T) {
	_, params := graphQueryToCypher(GraphQuery{
		StartNodeType:  "Doc",
		ReturnFields:   []string{"id"},
		ACLTokensParam: "acl_tokens",
	}, map[string]any{"acl_tokens": []string{"tenant:app"}})

	tokens, ok := params["acl_tokens"].([]string)
	if !ok || len(tokens) != 1 || tokens[0] != "tenant:app" {
		t.Fatalf("params[acl_tokens] = %#v, want bound ACL tokens", params["acl_tokens"])
	}
}

func TestGraphQueryToCypherPreservesCustomParams(t *testing.T) {
	_, params := graphQueryToCypher(GraphQuery{
		StartNodeType: "Doc",
		ReturnFields:  []string{"id"},
		Strategy:      "finance_deep",
	}, map[string]any{"custom_depth": 8})

	if got := params["custom_depth"]; got != 8 {
		t.Fatalf("params[custom_depth] = %#v, want 8", got)
	}
}

func TestNeo4jGraphAdapterExecuteQuery(t *testing.T) {
	runner := &recordingCypherRunner{result: []map[string]any{{"id": "node-a"}}}
	adapter := NewNeo4jGraphAdapter(runner)

	results, err := adapter.ExecuteQuery(context.Background(), GraphQuery{
		StartNodeType: "Doc",
		ReturnFields:  []string{"id"},
	}, map[string]any{"acl_tokens": []string{"tenant:app"}})
	if err != nil {
		t.Fatalf("ExecuteQuery() error = %v", err)
	}
	if len(results) != 1 || results[0]["id"] != "node-a" {
		t.Fatalf("results = %#v, want node-a", results)
	}
	if !strings.Contains(runner.cypher, "MATCH (n0:Doc)") {
		t.Fatalf("runner cypher = %q, want query execution", runner.cypher)
	}
}
