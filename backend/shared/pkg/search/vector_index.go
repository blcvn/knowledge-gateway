package search

import (
    "fmt"
    "math"
    "sort"
    "sync"
)

var ErrDimensionMismatch = fmt.Errorf("vector dimension mismatch")

type VectorIndex struct {
    mu       sync.RWMutex
    vectors  map[string][]float32   // docID → embedding
    sessions map[string]string      // docID → sessionID
    dims     int
    Dirty    bool
}

func NewVectorIndex(dims int) *VectorIndex {
    if dims <= 0 { dims = 384 }
    return &VectorIndex{
        vectors:  make(map[string][]float32),
        sessions: make(map[string]string),
        dims:     dims,
    }
}

func (v *VectorIndex) Add(docID, sessionID string, vec []float32) error {
    if len(vec) != v.dims { return ErrDimensionMismatch }
    v.mu.Lock()
    v.vectors[docID] = vec
    v.sessions[docID] = sessionID
    v.Dirty = true
    v.mu.Unlock()
    return nil
}

func (v *VectorIndex) Remove(docID string) {
    v.mu.Lock()
    delete(v.vectors, docID)
    delete(v.sessions, docID)
    v.Dirty = true
    v.mu.Unlock()
}

func (v *VectorIndex) Search(query []float32, limit int) []VectorResult {
    if len(query) != v.dims { return nil }
    v.mu.RLock()
    defer v.mu.RUnlock()

    scored := make([]VectorResult, 0, len(v.vectors))
    for docID, vec := range v.vectors {
        score := cosineSimilarity(query, vec)
        scored = append(scored, VectorResult{DocID: docID, SessionID: v.sessions[docID], Score: score})
    }
    sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
    if limit > len(scored) { limit = len(scored) }
    return scored[:limit]
}

func (v *VectorIndex) DocCount() int {
    v.mu.RLock()
    defer v.mu.RUnlock()
    return len(v.vectors)
}

func cosineSimilarity(a, b []float32) float64 {
    var dot, normA, normB float64
    for i := range a {
        dot  += float64(a[i]) * float64(b[i])
        normA += float64(a[i]) * float64(a[i])
        normB += float64(b[i]) * float64(b[i])
    }
    if normA == 0 || normB == 0 { return 0 }
    return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
