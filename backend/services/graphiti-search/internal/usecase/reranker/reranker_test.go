package reranker

import (
	"context"
	"testing"

	"vnp-memory/services/graphiti-search/internal/domain"
)

func TestRRFReranker(t *testing.T) {
	r := NewRRFReranker(60)
	res := []domain.SearchResult{
		{EntityID: "1", Score: 0.9, MethodUsed: domain.MethodCosine},
		{EntityID: "1", Score: 0.8, MethodUsed: domain.MethodBM25},
		{EntityID: "2", Score: 0.85, MethodUsed: domain.MethodCosine},
	}
	ranked, err := r.Rerank(context.Background(), "test", res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranked) != 2 {
		t.Fatalf("expected 2 ranked results, got %d", len(ranked))
	}
	if ranked[0].EntityID != "1" {
		t.Errorf("expected entity 1 to be ranked first")
	}
}
