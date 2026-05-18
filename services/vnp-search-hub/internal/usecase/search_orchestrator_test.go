package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
)

// mockEngine implements EngineSearcher for testing.
type mockEngine struct {
	name    string
	results []SearchResult
	fail    bool
}

func (m *mockEngine) Name() string { return m.name }

func (m *mockEngine) Search(_ context.Context, query string, limit int) ([]SearchResult, error) {
	if m.fail {
		return nil, fmt.Errorf("engine %s unavailable", m.name)
	}
	var results []SearchResult
	for _, r := range m.results {
		results = append(results, r)
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

func TestSearchOrchestrator_AllEngines(t *testing.T) {
	engines := map[string]EngineSearcher{
		"cognee": &mockEngine{name: "cognee", results: []SearchResult{
			{ID: "c1", Engine: "cognee", Type: "semantic", Content: "test", Score: 0.95},
		}},
		"graphiti": &mockEngine{name: "graphiti", results: []SearchResult{
			{ID: "g1", Engine: "graphiti", Type: "episodic", Content: "test", Score: 0.85},
		}},
		"zep": &mockEngine{name: "zep", results: []SearchResult{
			{ID: "z1", Engine: "zep", Type: "conversational", Content: "test", Score: 0.90},
		}},
	}

	orch := NewSearchOrchestrator(engines, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	resp, err := orch.Search(context.Background(), SearchRequest{
		Query: "user preferences",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if resp.Total != 3 {
		t.Errorf("expected 3 results, got %d", resp.Total)
	}
	if resp.Facets.ByEngine["cognee"] != 1 {
		t.Errorf("expected 1 cognee result, got %d", resp.Facets.ByEngine["cognee"])
	}
	// Results should be sorted by score descending
	if resp.Results[0].Score < resp.Results[1].Score {
		t.Error("results not sorted by score descending")
	}
}

func TestSearchOrchestrator_PartialFailure(t *testing.T) {
	engines := map[string]EngineSearcher{
		"cognee": &mockEngine{name: "cognee", results: []SearchResult{
			{ID: "c1", Engine: "cognee", Type: "semantic", Content: "test", Score: 0.95},
		}},
		"graphiti": &mockEngine{name: "graphiti", fail: true}, // This engine fails
	}

	orch := NewSearchOrchestrator(engines, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	resp, err := orch.Search(context.Background(), SearchRequest{
		Query: "test",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("Search should not fail on partial engine failure: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected 1 result from healthy engine, got %d", resp.Total)
	}
}

func TestSearchOrchestrator_SelectiveEngines(t *testing.T) {
	engines := map[string]EngineSearcher{
		"cognee":   &mockEngine{name: "cognee", results: []SearchResult{{ID: "c1", Engine: "cognee", Score: 0.9}}},
		"graphiti": &mockEngine{name: "graphiti", results: []SearchResult{{ID: "g1", Engine: "graphiti", Score: 0.8}}},
		"zep":      &mockEngine{name: "zep", results: []SearchResult{{ID: "z1", Engine: "zep", Score: 0.7}}},
	}

	orch := NewSearchOrchestrator(engines, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	resp, err := orch.Search(context.Background(), SearchRequest{
		Query:   "test",
		Engines: []string{"cognee", "zep"}, // Only 2 of 3
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected 2 results, got %d", resp.Total)
	}
	if _, ok := resp.Facets.ByEngine["graphiti"]; ok {
		t.Error("graphiti should not be in results when not requested")
	}
}

func TestSearchOrchestrator_DefaultLimit(t *testing.T) {
	engines := map[string]EngineSearcher{
		"cognee": &mockEngine{name: "cognee", results: []SearchResult{{ID: "c1", Engine: "cognee", Score: 0.9}}},
	}

	orch := NewSearchOrchestrator(engines, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	resp, err := orch.Search(context.Background(), SearchRequest{Query: "test", Limit: 0})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected 1 result, got %d", resp.Total)
	}
}
