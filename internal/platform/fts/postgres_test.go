package fts

import (
	"strings"
	"testing"
)

func TestBuildTsQueryExpr(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantSQL string
		wantArg string
	}{
		{name: "all_tokens", mode: "all_tokens", wantSQL: "plainto_tsquery('simple', $1)", wantArg: "payment gateway"},
		{name: "any_token", mode: "any_token", wantSQL: "to_tsquery('simple', $1)", wantArg: "payment | gateway"},
		{name: "phrase", mode: "phrase", wantSQL: "phraseto_tsquery('simple', $1)", wantArg: "payment gateway"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sql, arg := buildTsQueryExpr("simple", FTSQuery{Text: "payment gateway", Mode: tc.mode})
			if sql != tc.wantSQL {
				t.Fatalf("sql = %q, want %q", sql, tc.wantSQL)
			}
			if arg != tc.wantArg {
				t.Fatalf("arg = %q, want %q", arg, tc.wantArg)
			}
		})
	}
}

func TestBuildDocumentVectorExpr(t *testing.T) {
	if got := buildDocumentVectorExpr("simple", nil); got != "fts_vector" {
		t.Fatalf("buildDocumentVectorExpr(nil) = %q, want fts_vector", got)
	}

	got := buildDocumentVectorExpr("simple", []string{"title", "body", "status_value"})
	if !strings.Contains(got, "properties ->> 'title'") {
		t.Fatalf("vector expr missing title weighting: %q", got)
	}
	if !strings.Contains(got, "properties ->> 'body'") {
		t.Fatalf("vector expr missing body weighting: %q", got)
	}
	if !strings.Contains(got, "status_value::text") {
		t.Fatalf("vector expr missing built-in field handling: %q", got)
	}
}

func TestBuildSearchStatement(t *testing.T) {
	adapter := NewPgFTSAdapter(nil, "")
	stmt, args := adapter.buildSearchStatement(
		FTSQuery{Text: "payment gateway", Mode: "any_token", Fields: []string{"title"}},
		FTSFilter{DomainIDs: []string{"d1"}, OwnerTenantIDs: []string{"t1"}},
		SearchOptions{TopK: 20},
	)

	wantContains := []string{
		"to_tsquery('simple', $1)",
		"properties ->> 'title'",
		"domain_id = ANY($2)",
		"owner_tenant_id = ANY($3)",
		"LIMIT $4",
	}
	for _, want := range wantContains {
		if !strings.Contains(stmt, want) {
			t.Fatalf("statement missing %q: %s", want, stmt)
		}
	}
	if len(args) != 4 {
		t.Fatalf("args len = %d, want 4", len(args))
	}
	if args[0] != "payment | gateway" {
		t.Fatalf("args[0] = %v, want joined OR query", args[0])
	}
}
