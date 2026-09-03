package search

import (
	"context"
	"testing"
)

type fakeVectorRetriever struct {
	results []Result
	err     error
}

func (f *fakeVectorRetriever) Search(ctx context.Context, namespace, query string, topK int) ([]Result, error) {
	return f.results, f.err
}

type fakeTextRetriever struct {
	results []Result
	err     error
}

func (f *fakeTextRetriever) Search(ctx context.Context, namespace, query string, topK int) ([]Result, error) {
	return f.results, f.err
}

type fakeCentrality struct {
	scores map[string]float64
	err    error
}

func (f *fakeCentrality) Scores(ctx context.Context, namespace string, nodeIDs []string) (map[string]float64, error) {
	return f.scores, f.err
}

func TestEngineHybridSearch(t *testing.T) {
	engine := &Engine{
		vector: &fakeVectorRetriever{
			results: []Result{
				{ID: "n1", Label: "Requirement", Score: 0.9, Properties: map[string]any{"confidence": 0.9}},
				{ID: "n2", Label: "UseCase", Score: 0.7, Properties: map[string]any{"confidence": 0.8}},
			},
		},
		text: &fakeTextRetriever{
			results: []Result{
				{ID: "n1", Label: "Requirement", Score: 0.6, Properties: map[string]any{"confidence": 0.9}},
				{ID: "n3", Label: "Requirement", Score: 0.8, Properties: map[string]any{"confidence": 0.95}},
			},
		},
		centrality: &fakeCentrality{
			scores: map[string]float64{
				"n1": 10,
				"n2": 1,
				"n3": 5,
			},
		},
	}

	got, err := engine.HybridSearch(context.Background(), "graph/app/tenant", "payment", Options{
		TopK:          2,
		Alpha:         0.5,
		Beta:          0.2,
		EntityTypes:   []string{"Requirement"},
		MinConfidence: 0.85,
	})
	if err != nil {
		t.Fatalf("HybridSearch error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
	if got[0].ID != "n1" {
		t.Fatalf("expected n1 top ranked, got %s", got[0].ID)
	}
}
