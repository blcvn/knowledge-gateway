package reranker

import (
	"context"
	"testing"

	"vnp-memory/services/graphiti-search/internal/domain"
)

func TestMMRReranker_Rerank(t *testing.T) {
	reranker := NewMMRReranker(0.5)

	// 1 query, 3 results with various scores and content
	query := "blockchain network"
	results := []domain.SearchResult{
		{EntityID: "1", Score: 0.9, Content: "blockchain network architecture"},
		{EntityID: "2", Score: 0.85, Content: "blockchain network architecture"}, // Very similar to 1 -> penalty
		{EntityID: "3", Score: 0.7, Content: "consensus algorithm used in nodes"}, // Different content -> low penalty
	}

	reranked, err := reranker.Rerank(context.Background(), query, results)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(reranked) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(reranked))
	}

	// Entity 1 should be first (highest score)
	if reranked[0].EntityID != "1" {
		t.Errorf("Expected first entity to be 1, got %s", reranked[0].EntityID)
	}

	// Entity 3 should be ranked higher than 2 because 2 is penalized heavily for being too similar to 1
	if reranked[1].EntityID != "3" {
		t.Errorf("Expected second entity to be 3 (MMR penalty applied to 2), got %s", reranked[1].EntityID)
	}
}
