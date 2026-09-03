package search_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vnp-memory/pkg/search"
)

func TestBM25_AddAndSearch(t *testing.T) {
	idx := search.NewBM25Index()
	idx.Add("doc1", "s1", "a1", "t1", "Go programming language goroutines")
	idx.Add("doc2", "s1", "a1", "t1", "Python machine learning")
	results := idx.Search("goroutines", 5)
	assert.NotEmpty(t, results)
	assert.Equal(t, "doc1", results[0].DocID)
}

func TestBM25_Remove(t *testing.T) {
	idx := search.NewBM25Index()
	idx.Add("doc1", "s1", "a1", "t1", "Go programming language")
	idx.Add("doc2", "s1", "a1", "t1", "Python machine learning")
	idx.Remove("doc1")
	results := idx.Search("Go programming", 5)
	for _, r := range results {
		assert.NotEqual(t, "doc1", r.DocID)
	}
}

func TestBM25_DocCount(t *testing.T) {
	idx := search.NewBM25Index()
	assert.Equal(t, 0, idx.DocCount())
	idx.Add("doc1", "s1", "a1", "t1", "text one")
	idx.Add("doc2", "s1", "a1", "t1", "text two")
	assert.Equal(t, 2, idx.DocCount())
}

func TestBM25_SurviveRestart(t *testing.T) {
	dir := t.TempDir()
	idx := search.NewBM25Index()
	idx.Add("doc1", "s1", "a1", "t1", "golang test")
	p := search.NewIndexPersister(idx, search.NewVectorIndex(384), dir)
	// Trigger immediate save (bypass debounce)
	p.Schedule()
	time.Sleep(50 * time.Millisecond)

	idx2 := search.NewBM25Index()
	p2 := search.NewIndexPersister(idx2, search.NewVectorIndex(384), dir)
	p2.LoadAsync()
	time.Sleep(100 * time.Millisecond)
	results := idx2.Search("golang", 5)
	assert.NotEmpty(t, results)
}

func TestTokenize_CJK(t *testing.T) {
	tokens := search.Tokenize("开放源代码")
	// Should produce CJK bigrams
	assert.NotEmpty(t, tokens)
}

func TestTokenize_Stemming(t *testing.T) {
	tokens := search.Tokenize("programming goroutines")
	assert.Contains(t, tokens, "goroutin") // "goroutines" → "goroutin" (strip 'es')
}
