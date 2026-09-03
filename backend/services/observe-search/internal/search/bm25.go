package search

import pkg_search "github.com/vnp-memory/pkg/search"

// BM25SearchResult is an alias for the pkg-level type.
type BM25SearchResult = pkg_search.BM25Result

// NewBM25Index creates a new BM25 in-memory index.
func NewBM25Index() *pkg_search.BM25Index {
    return pkg_search.NewBM25Index()
}
