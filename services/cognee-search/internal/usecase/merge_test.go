package usecase

import (
	"testing"
	"vnp-memory/services/cognee-search/internal/domain"
)

func TestMerge_RRF(t *testing.T) {
	// Simple test for RRF logic
	results := []domain.SearchResult{
		{ID: "1", Content: "A", Score: 0.9, Strategy: domain.Similarity},
		{ID: "2", Content: "A", Score: 0.8, Strategy: domain.Chunks},
		{ID: "3", Content: "B", Score: 0.7, Strategy: domain.Similarity},
	}

	merged := merge(results, 10)

	if len(merged) != 2 {
		t.Errorf("Expected 2 merged results, got %d", len(merged))
	}

	if merged[0].Content != "A" {
		t.Errorf("Expected 'A' to be ranked first due to RRF fusion")
	}
}
