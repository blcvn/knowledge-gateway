package conformance

import (
	"context"
	"testing"
	"time"

	"kg-service/internal/platform/fts"
	"kg-service/internal/platform/graphstore"
	"kg-service/internal/platform/vector"
	"kg-service/internal/platform/vectorstore"
)

func AssertEmbeddingProviderConformance(t *testing.T, provider vector.EmbeddingProvider) {
	t.Helper()

	if provider == nil {
		t.Fatal("provider is nil")
	}
	if provider.Dimensions() <= 0 {
		t.Fatalf("Dimensions() = %d, want positive", provider.Dimensions())
	}
	if provider.ModelID() == "" {
		t.Fatal("ModelID() is empty")
	}
	first, err := provider.Embed(context.Background(), "conformance payload")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	second, err := provider.Embed(context.Background(), "conformance payload")
	if err != nil {
		t.Fatalf("Embed() second error = %v", err)
	}
	if len(first) != provider.Dimensions() {
		t.Fatalf("embedding len = %d, want %d", len(first), provider.Dimensions())
	}
	if len(second) != len(first) {
		t.Fatalf("embedding len mismatch = %d vs %d", len(second), len(first))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("embedding is not deterministic at %d: %v vs %v", i, first, second)
		}
	}
}

func AssertVectorAdapterConformance(t *testing.T, adapter vectorstore.VectorAdapter) {
	t.Helper()

	if adapter == nil {
		t.Fatal("adapter is nil")
	}
	must := func(err error) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	must(adapter.Upsert(context.Background(), vectorstore.VectorDocument{
		NodeID:        "a",
		NodeType:      "Doc",
		DomainID:      "d",
		OwnerTenantID: "t",
		OwnerAppID:    "a",
		ACLVisibleTo:  []string{"t:a"},
		SyncVersion:   1,
		Embedding:     []float64{1, 0},
	}))
	must(adapter.Upsert(context.Background(), vectorstore.VectorDocument{
		NodeID:        "b",
		NodeType:      "Doc",
		DomainID:      "d",
		OwnerTenantID: "t",
		OwnerAppID:    "a",
		ACLVisibleTo:  []string{"t:a"},
		Embedding:     []float64{0, 1},
	}))
	results, err := adapter.ANN(context.Background(), []float64{1, 0}, vectorstore.VectorFilter{
		DomainIDs:    []string{"d"},
		ACLVisibleTo: []string{"t:a"},
	}, vectorstore.ANNOptions{TopK: 1})
	if err != nil {
		t.Fatalf("ANN() error = %v", err)
	}
	if len(results) != 1 || results[0].Document.NodeID != "a" {
		t.Fatalf("ANN results = %#v, want node a", results)
	}
	snapshot, err := adapter.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if len(snapshot) == 0 {
		t.Fatal("Snapshot() returned no documents")
	}
	version, err := adapter.ReadSyncVersion(context.Background(), "a")
	if err != nil {
		t.Fatalf("ReadSyncVersion() error = %v", err)
	}
	if version == 0 {
		t.Fatal("ReadSyncVersion() returned zero version")
	}
	must(adapter.Delete(context.Background(), "a"))
	results, err = adapter.ANN(context.Background(), []float64{1, 0}, vectorstore.VectorFilter{DomainIDs: []string{"d"}}, vectorstore.ANNOptions{TopK: 10})
	if err != nil {
		t.Fatalf("ANN() after delete error = %v", err)
	}
	for _, result := range results {
		if result.Document.NodeID == "a" {
			t.Fatalf("deleted document still present: %#v", results)
		}
	}
}

func AssertFTSAdapterConformance(t *testing.T, adapter fts.FTSAdapter) {
	t.Helper()

	if adapter == nil {
		t.Fatal("adapter is nil")
	}
	now := time.Date(2026, time.June, 18, 7, 0, 0, 0, time.UTC)
	must := func(err error) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	must(adapter.Index(context.Background(), fts.FTSDocument{
		NodeID:        "a",
		NodeType:      "Doc",
		DomainID:      "d",
		OwnerTenantID: "t",
		OwnerAppID:    "a",
		ACLVisibleTo:  []string{"t:a"},
		Fields: map[string]string{
			"title": "payment gateway",
			"body":  "other text",
		},
		CreatedAt: now,
	}))
	must(adapter.Index(context.Background(), fts.FTSDocument{
		NodeID:        "b",
		NodeType:      "Doc",
		DomainID:      "d",
		OwnerTenantID: "t",
		OwnerAppID:    "a",
		ACLVisibleTo:  []string{"t:a"},
		Fields: map[string]string{
			"title": "payment gateway",
			"body":  "open source tools",
		},
		CreatedAt: now.Add(time.Minute),
	}))
	allTokens, err := adapter.Search(context.Background(), fts.FTSQuery{Text: "payment gateway", Mode: "all_tokens"}, fts.FTSFilter{DomainIDs: []string{"d"}, ACLVisibleTo: []string{"t:a"}}, fts.SearchOptions{TopK: 10})
	if err != nil {
		t.Fatalf("Search(all_tokens) error = %v", err)
	}
	if len(allTokens) != 2 {
		t.Fatalf("all_tokens results len = %d, want 2", len(allTokens))
	}
	anyToken, err := adapter.Search(context.Background(), fts.FTSQuery{Text: "gateway missing", Mode: "any_token"}, fts.FTSFilter{DomainIDs: []string{"d"}, ACLVisibleTo: []string{"t:a"}}, fts.SearchOptions{TopK: 10})
	if err != nil {
		t.Fatalf("Search(any_token) error = %v", err)
	}
	if len(anyToken) != 2 {
		t.Fatalf("any_token results len = %d, want 2", len(anyToken))
	}
	phrase, err := adapter.Search(context.Background(), fts.FTSQuery{Text: "payment gateway", Mode: "phrase"}, fts.FTSFilter{DomainIDs: []string{"d"}, ACLVisibleTo: []string{"t:a"}}, fts.SearchOptions{TopK: 10})
	if err != nil {
		t.Fatalf("Search(phrase) error = %v", err)
	}
	if len(phrase) != 2 {
		t.Fatalf("phrase results len = %d, want 2", len(phrase))
	}
	fieldRestricted, err := adapter.Search(context.Background(), fts.FTSQuery{Text: "payment gateway", Mode: "all_tokens", Fields: []string{"title"}}, fts.FTSFilter{DomainIDs: []string{"d"}, ACLVisibleTo: []string{"t:a"}}, fts.SearchOptions{TopK: 10})
	if err != nil {
		t.Fatalf("Search(fields) error = %v", err)
	}
	if len(fieldRestricted) != 2 {
		t.Fatalf("field restricted results len = %d, want 2", len(fieldRestricted))
	}
	must(adapter.Delete(context.Background(), "a"))
	results, err := adapter.Search(context.Background(), fts.FTSQuery{Text: "payment gateway", Mode: "all_tokens"}, fts.FTSFilter{DomainIDs: []string{"d"}}, fts.SearchOptions{TopK: 10})
	if err != nil {
		t.Fatalf("Search(delete) error = %v", err)
	}
	if len(results) != 1 || results[0].Document.NodeID != "b" {
		t.Fatalf("delete results = %#v, want b", results)
	}
}

func AssertGraphAdapterConformance(t *testing.T, adapter graphstore.GraphAdapter) {
	t.Helper()

	if adapter == nil {
		t.Fatal("adapter is nil")
	}
	must := func(err error) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	must(adapter.UpsertNode(context.Background(), graphstore.GraphNode{
		ID:            "a",
		NodeType:      "Doc",
		DomainID:      "d",
		OwnerTenantID: "t",
		OwnerAppID:    "a",
		ACLVisibleTo:  []string{"t:a"},
		SyncVersion:   1,
		Properties: map[string]any{
			"title": "alpha",
		},
		CreatedAt: time.Date(2026, time.June, 18, 7, 0, 0, 0, time.UTC),
	}))
	must(adapter.UpsertNode(context.Background(), graphstore.GraphNode{
		ID:            "b",
		NodeType:      "Doc",
		DomainID:      "d",
		OwnerTenantID: "t",
		OwnerAppID:    "a",
		ACLVisibleTo:  []string{"t:a"},
		Properties: map[string]any{
			"title": "beta",
		},
		CreatedAt: time.Date(2026, time.June, 18, 7, 1, 0, 0, time.UTC),
	}))
	must(adapter.UpsertRelationship(context.Background(), graphstore.GraphRelationship{
		ID:         "r1",
		RelType:    "LINKS",
		FromNodeID: "a",
		ToNodeID:   "b",
		DomainID:   "d",
	}))
	results, err := adapter.ExecuteQuery(context.Background(), graphstore.GraphQuery{
		StartNodeType:  "Doc",
		StartMatch:     map[string]any{"title": "alpha"},
		Hops:           []graphstore.GraphQueryHop{{RelType: "LINKS", ToNodeType: "Doc", Direction: "out"}},
		ReturnFields:   []string{"id"},
		ACLTokensParam: "acl_tokens",
	}, map[string]any{"acl_tokens": []string{"t:a"}})
	if err != nil {
		t.Fatalf("ExecuteQuery() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("ExecuteQuery() results len = %d, want 1", len(results))
	}
	if got := results[0]["id"]; got != "a" {
		t.Fatalf("ExecuteQuery() id = %#v, want a", got)
	}
	version, err := adapter.ReadSyncVersion(context.Background(), "a")
	if err != nil {
		t.Fatalf("ReadSyncVersion() error = %v", err)
	}
	if version == 0 {
		t.Fatal("ReadSyncVersion() returned zero version")
	}
	must(adapter.DeleteNode(context.Background(), "b"))
	results, err = adapter.ExecuteQuery(context.Background(), graphstore.GraphQuery{
		StartNodeType:  "Doc",
		StartMatch:     map[string]any{"title": "alpha"},
		Hops:           []graphstore.GraphQueryHop{{RelType: "LINKS", ToNodeType: "Doc", Direction: "out"}},
		ReturnFields:   []string{"id"},
		ACLTokensParam: "acl_tokens",
	}, map[string]any{"acl_tokens": []string{"t:a"}})
	if err != nil {
		t.Fatalf("ExecuteQuery() after delete error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("results after delete = %#v, want none", results)
	}
}
