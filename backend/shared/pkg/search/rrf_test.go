package search_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vnp-memory/pkg/search"
)

func TestRRFFuse_CombinesScores(t *testing.T) {
	bm25r := []search.BM25Result{{DocID: "d1", Score: 2.5}, {DocID: "d2", Score: 1.5}}
	vecr := []search.VectorResult{{DocID: "d2", Score: 0.9}, {DocID: "d1", Score: 0.7}}
	results := search.RRFFuse(bm25r, vecr, nil, search.DefaultWeights, 5)
	assert.Equal(t, 2, len(results))
	// Both docs should appear in results
	docIDs := []string{results[0].DocID, results[1].DocID}
	assert.Contains(t, docIDs, "d1")
	assert.Contains(t, docIDs, "d2")
}

func TestRRFFuse_OnlyBM25(t *testing.T) {
	bm25r := []search.BM25Result{
		{DocID: "d1", SessionID: "s1", Score: 3.0},
		{DocID: "d2", SessionID: "s1", Score: 1.0},
	}
	weights := search.ScoreWeights{BM25: 1.0, Vector: 0.0}
	results := search.RRFFuse(bm25r, nil, nil, weights, 5)
	assert.Equal(t, 2, len(results))
	assert.Equal(t, "d1", results[0].DocID)
}

func TestRRFFuse_Limit(t *testing.T) {
	bm25r := []search.BM25Result{
		{DocID: "d1", Score: 3.0},
		{DocID: "d2", Score: 2.0},
		{DocID: "d3", Score: 1.0},
	}
	results := search.RRFFuse(bm25r, nil, nil, search.DefaultWeights, 2)
	assert.Equal(t, 2, len(results))
}

func TestRRFFuse_GraphBoost(t *testing.T) {
	bm25r := []search.BM25Result{{DocID: "d1", Score: 2.5}}
	vecr := []search.VectorResult{{DocID: "d2", Score: 0.9}}
	graphr := []search.GraphResult{{DocID: "d2", Score: 1.0}}
	weights := search.ScoreWeights{BM25: 0.3, Vector: 0.5, Graph: 0.2}
	results := search.RRFFuse(bm25r, vecr, graphr, weights, 5)
	assert.Equal(t, 2, len(results))
}
