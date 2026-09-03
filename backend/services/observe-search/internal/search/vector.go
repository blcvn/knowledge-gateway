package search

import pkg_search "github.com/vnp-memory/pkg/search"

// NewVectorIndex creates a new dense vector cosine index.
func NewVectorIndex(dims int) *pkg_search.VectorIndex {
    return pkg_search.NewVectorIndex(dims)
}
