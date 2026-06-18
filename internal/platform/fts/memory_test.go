package fts

import (
	"context"
	"testing"
)

func TestInMemoryFTSAdapterSearchModes(t *testing.T) {
	adapter := NewInMemoryFTSAdapter()
	must := func(err error) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	must(adapter.Index(context.Background(), FTSDocument{
		NodeID:       "a",
		DomainID:     "d",
		ACLVisibleTo: []string{"t:a"},
		Fields: map[string]string{
			"title": "Open search service",
			"body":  "Fast and reliable indexing",
		},
	}))
	must(adapter.Index(context.Background(), FTSDocument{
		NodeID:       "b",
		DomainID:     "d",
		ACLVisibleTo: []string{"t:a"},
		Fields: map[string]string{
			"title": "Search service",
			"body":  "Open source tools",
		},
	}))

	allTokens, err := adapter.Search(context.Background(), FTSQuery{Text: "open reliable", Mode: "all_tokens"}, FTSFilter{DomainIDs: []string{"d"}, ACLVisibleTo: []string{"t:a"}}, SearchOptions{TopK: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(allTokens) != 1 || allTokens[0].Document.NodeID != "a" {
		t.Fatalf("all_tokens results = %#v, want a", allTokens)
	}

	anyToken, err := adapter.Search(context.Background(), FTSQuery{Text: "reliable missing", Mode: "any_token"}, FTSFilter{DomainIDs: []string{"d"}, ACLVisibleTo: []string{"t:a"}}, SearchOptions{TopK: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(anyToken) != 1 || anyToken[0].Document.NodeID != "a" {
		t.Fatalf("any_token results = %#v, want a", anyToken)
	}

	phrase, err := adapter.Search(context.Background(), FTSQuery{Text: "fast and reliable", Mode: "phrase"}, FTSFilter{DomainIDs: []string{"d"}, ACLVisibleTo: []string{"t:a"}}, SearchOptions{TopK: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(phrase) != 1 || phrase[0].Document.NodeID != "a" {
		t.Fatalf("phrase results = %#v, want a", phrase)
	}
}

func TestInMemoryFTSAdapterFieldRestrictionAndDelete(t *testing.T) {
	adapter := NewInMemoryFTSAdapter()
	_ = adapter.Index(context.Background(), FTSDocument{
		NodeID:       "a",
		DomainID:     "d",
		ACLVisibleTo: []string{"t:a"},
		Fields: map[string]string{
			"title": "Search service",
			"body":  "Open source tools",
		},
	})
	_ = adapter.Index(context.Background(), FTSDocument{
		NodeID:       "b",
		DomainID:     "d",
		ACLVisibleTo: []string{"t:a"},
		Fields: map[string]string{
			"title": "Another title",
			"body":  "Search service only in body",
		},
	})

	results, err := adapter.Search(context.Background(), FTSQuery{Text: "search service", Mode: "all_tokens", Fields: []string{"title"}}, FTSFilter{DomainIDs: []string{"d"}, ACLVisibleTo: []string{"t:a"}}, SearchOptions{TopK: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 || results[0].Document.NodeID != "a" {
		t.Fatalf("field-restricted results = %#v, want a", results)
	}

	_ = adapter.Delete(context.Background(), "a")
	results, err = adapter.Search(context.Background(), FTSQuery{Text: "search service", Mode: "all_tokens"}, FTSFilter{DomainIDs: []string{"d"}}, SearchOptions{TopK: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 || results[0].Document.NodeID != "b" {
		t.Fatalf("delete results = %#v, want b", results)
	}
}
