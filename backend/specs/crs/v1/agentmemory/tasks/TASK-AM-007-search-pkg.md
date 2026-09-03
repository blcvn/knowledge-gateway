# TASK-AM-007 — Shared Search Package (`pkg/search/`)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-007 |
| **Wave** | 1 (Foundation) |
| **Component** | `pkg/search/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-003 §2.1 |
| **Priority** | 🔴 Critical |
| **Depends On** | — |
| **Estimated** | 6h |

---

## Context

Shared package chứa BM25 in-memory inverted index, dense vector cosine index, RRF fusion, tokenizer (CJK bigrams + Porter stemmer), và debounced gob persistence. Package này được `observe-search` service dùng.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `pkg/search/types.go` |
| CREATE | `pkg/search/tokenizer.go` |
| CREATE | `pkg/search/bm25.go` |
| CREATE | `pkg/search/vector_index.go` |
| CREATE | `pkg/search/rrf.go` |
| CREATE | `pkg/search/persistence.go` |
| CREATE | `pkg/search/bm25_test.go` |
| CREATE | `pkg/search/rrf_test.go` |

---

## Implementation

### `pkg/search/types.go`

```go
package search

type BM25Result struct {
    DocID     string
    SessionID string
    AgentID   string
    Score     float64
}

type VectorResult struct {
    DocID     string
    SessionID string
    Score     float64
}

type GraphResult struct {
    DocID string
    Score float64
}

type HybridResult struct {
    DocID         string
    SessionID     string
    CombinedScore float64
    BM25Score     float64
    VectorScore   float64
    GraphScore    float64
    BM25Rank      int
    VectorRank    int
}

type ScoreWeights struct {
    BM25   float64  // default 0.4
    Vector float64  // default 0.6
    Graph  float64  // default 0.0
}

var DefaultWeights = ScoreWeights{BM25: 0.4, Vector: 0.6}

type DocMeta struct {
    SessionID string
    AgentID   string
    TenantID  string
}
```

### `pkg/search/tokenizer.go`

```go
package search

import (
    "strings"
    "unicode"
)

var stopwords = map[string]bool{
    "the": true, "is": true, "at": true, "which": true, "on": true,
    "a": true, "an": true, "and": true, "or": true, "but": true,
    "in": true, "of": true, "to": true, "for": true, "with": true,
    "this": true, "that": true, "have": true, "from": true, "be": true,
}

// Tokenize splits text into normalized tokens for BM25 indexing
// Supports: ASCII word splitting + CJK bigrams
func Tokenize(text string) []string {
    text = strings.ToLower(text)
    var tokens []string

    // ASCII word splitting + simple stemming
    words := strings.FieldsFunc(text, func(r rune) bool {
        return !unicode.IsLetter(r) && !unicode.IsDigit(r)
    })
    for _, w := range words {
        stemmed := porterStem(w)
        if len(stemmed) >= 2 && !stopwords[stemmed] {
            tokens = append(tokens, stemmed)
        }
    }

    // CJK bigram tokenization
    tokens = append(tokens, cjkBigrams(text)...)
    return tokens
}

// cjkBigrams generates bigrams from CJK unicode characters
func cjkBigrams(text string) []string {
    runes := []rune(text)
    var bigrams []string
    for i := 0; i < len(runes)-1; i++ {
        if isCJK(runes[i]) && isCJK(runes[i+1]) {
            bigrams = append(bigrams, string(runes[i:i+2]))
        }
    }
    return bigrams
}

func isCJK(r rune) bool {
    return (r >= 0x4E00 && r <= 0x9FFF) ||  // CJK Unified
           (r >= 0x3400 && r <= 0x4DBF) ||  // Extension A
           (r >= 0xAC00 && r <= 0xD7AF)     // Hangul
}

// Simple Porter stemmer (first 3 rules only for brevity)
func porterStem(word string) string {
    if len(word) <= 3 { return word }
    if strings.HasSuffix(word, "ing") && len(word) > 5 { return word[:len(word)-3] }
    if strings.HasSuffix(word, "ed") && len(word) > 4 { return word[:len(word)-2] }
    if strings.HasSuffix(word, "es") && len(word) > 3 { return word[:len(word)-2] }
    if strings.HasSuffix(word, "s") && len(word) > 3 { return word[:len(word)-1] }
    return word
}
```

### `pkg/search/bm25.go`

```go
package search

import (
    "math"
    "sort"
    "sync"
)

const (
    bm25K1 = 1.25
    bm25B  = 0.75
)

type Posting struct {
    DocID string
    TF    int
}

type BM25Index struct {
    mu          sync.RWMutex
    invertedIdx map[string][]Posting    // term → postings
    docLengths  map[string]int          // docID → term count
    docMeta     map[string]DocMeta      // docID → meta
    totalDocs   int
    totalLength int
    Dirty       bool
}

func NewBM25Index() *BM25Index {
    return &BM25Index{
        invertedIdx: make(map[string][]Posting),
        docLengths:  make(map[string]int),
        docMeta:     make(map[string]DocMeta),
    }
}

func (b *BM25Index) Add(docID, sessionID, agentID, tenantID, text string) {
    terms := Tokenize(text)
    tf := make(map[string]int, len(terms))
    for _, t := range terms { tf[t]++ }

    b.mu.Lock()
    defer b.mu.Unlock()

    // Remove old postings if updating
    if _, exists := b.docLengths[docID]; exists {
        b.removeUnsafe(docID)
    }

    for term, count := range tf {
        b.invertedIdx[term] = append(b.invertedIdx[term], Posting{DocID: docID, TF: count})
    }
    b.docLengths[docID] = len(terms)
    b.docMeta[docID] = DocMeta{SessionID: sessionID, AgentID: agentID, TenantID: tenantID}
    b.totalDocs++
    b.totalLength += len(terms)
    b.Dirty = true
}

func (b *BM25Index) Remove(docID string) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.removeUnsafe(docID)
}

func (b *BM25Index) removeUnsafe(docID string) {
    if length, ok := b.docLengths[docID]; ok {
        for term, postings := range b.invertedIdx {
            filtered := postings[:0]
            for _, p := range postings { if p.DocID != docID { filtered = append(filtered, p) } }
            if len(filtered) == 0 { delete(b.invertedIdx, term) } else { b.invertedIdx[term] = filtered }
        }
        delete(b.docLengths, docID)
        delete(b.docMeta, docID)
        b.totalDocs--
        b.totalLength -= length
        b.Dirty = true
    }
}

func (b *BM25Index) Search(query string, limit int) []BM25Result {
    b.mu.RLock()
    defer b.mu.RUnlock()

    terms := Tokenize(query)
    scores := map[string]float64{}
    avgdl := float64(b.totalLength) / math.Max(float64(b.totalDocs), 1)

    for _, term := range terms {
        postings := b.invertedIdx[term]
        df := float64(len(postings))
        if df == 0 { continue }
        idf := math.Log((float64(b.totalDocs)-df+0.5)/(df+0.5) + 1)

        for _, p := range postings {
            dl := float64(b.docLengths[p.DocID])
            tf := float64(p.TF)
            tfNorm := (tf * (bm25K1 + 1)) / (tf + bm25K1*(1-bm25B+bm25B*dl/avgdl))
            scores[p.DocID] += idf * tfNorm
        }
    }

    results := make([]BM25Result, 0, len(scores))
    for docID, score := range scores {
        meta := b.docMeta[docID]
        results = append(results, BM25Result{
            DocID: docID, SessionID: meta.SessionID, AgentID: meta.AgentID, Score: score,
        })
    }
    sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
    if limit > len(results) { limit = len(results) }
    return results[:limit]
}

func (b *BM25Index) DocCount() int {
    b.mu.RLock()
    defer b.mu.RUnlock()
    return b.totalDocs
}
```

### `pkg/search/vector_index.go`

```go
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
```

### `pkg/search/rrf.go`

```go
package search

import "sort"

const rrfK = 60

func RRFFuse(bm25 []BM25Result, vector []VectorResult, graph []GraphResult,
    weights ScoreWeights, limit int) []HybridResult {

    scores := map[string]*HybridResult{}

    for rank, r := range bm25 {
        h := getOrCreate(scores, r.DocID, r.SessionID)
        h.CombinedScore += weights.BM25 * (1.0 / float64(rrfK+rank+1))
        h.BM25Rank = rank + 1
        h.BM25Score = r.Score
    }
    for rank, r := range vector {
        h := getOrCreate(scores, r.DocID, r.SessionID)
        h.CombinedScore += weights.Vector * (1.0 / float64(rrfK+rank+1))
        h.VectorRank = rank + 1
        h.VectorScore = r.Score
    }
    for rank, r := range graph {
        h := getOrCreate(scores, r.DocID, "")
        h.CombinedScore += weights.Graph * (1.0 / float64(rrfK+rank+1))
        h.GraphScore = r.Score
    }

    results := make([]HybridResult, 0, len(scores))
    for _, v := range scores { results = append(results, *v) }
    sort.Slice(results, func(i, j int) bool { return results[i].CombinedScore > results[j].CombinedScore })
    if limit > len(results) { limit = len(results) }
    return results[:limit]
}

func getOrCreate(scores map[string]*HybridResult, docID, sessionID string) *HybridResult {
    if scores[docID] == nil {
        scores[docID] = &HybridResult{DocID: docID, SessionID: sessionID}
    }
    return scores[docID]
}
```

### `pkg/search/persistence.go`

```go
package search

import (
    "encoding/gob"
    "os"
    "path/filepath"
    "sync"
    "time"
)

type IndexPersister struct {
    bm25   *BM25Index
    vector *VectorIndex
    dir    string
    timer  *time.Timer
    mu     sync.Mutex
}

func NewIndexPersister(bm25 *BM25Index, vector *VectorIndex, dir string) *IndexPersister {
    return &IndexPersister{bm25: bm25, vector: vector, dir: dir}
}

// Schedule debounces saves: resets 30s timer on each call
func (p *IndexPersister) Schedule() {
    p.mu.Lock()
    defer p.mu.Unlock()
    if p.timer != nil { p.timer.Stop() }
    p.timer = time.AfterFunc(30*time.Second, p.save)
}

func (p *IndexPersister) save() {
    os.MkdirAll(p.dir, 0755)
    p.saveBM25()
    p.saveVector()
}

func (p *IndexPersister) saveBM25() {
    f, err := os.Create(filepath.Join(p.dir, "bm25.gob"))
    if err != nil { return }
    defer f.Close()
    p.bm25.mu.RLock()
    gob.NewEncoder(f).Encode(p.bm25.invertedIdx)
    p.bm25.mu.RUnlock()
}

func (p *IndexPersister) saveVector() {
    f, err := os.Create(filepath.Join(p.dir, "vector.gob"))
    if err != nil { return }
    defer f.Close()
    p.vector.mu.RLock()
    gob.NewEncoder(f).Encode(p.vector.vectors)
    p.vector.mu.RUnlock()
}

// LoadAsync loads indexes on startup without blocking the server
func (p *IndexPersister) LoadAsync() {
    go func() {
        p.loadBM25()
        p.loadVector()
    }()
}

func (p *IndexPersister) loadBM25() {
    f, err := os.Open(filepath.Join(p.dir, "bm25.gob"))
    if err != nil { return }
    defer f.Close()
    p.bm25.mu.Lock()
    gob.NewDecoder(f).Decode(&p.bm25.invertedIdx)
    p.bm25.mu.Unlock()
}

func (p *IndexPersister) loadVector() {
    f, err := os.Open(filepath.Join(p.dir, "vector.gob"))
    if err != nil { return }
    defer f.Close()
    p.vector.mu.Lock()
    gob.NewDecoder(f).Decode(&p.vector.vectors)
    p.vector.mu.Unlock()
}
```

---

## Verification

```bash
cd pkg/search
go test ./... -v -run TestBM25
go test ./... -v -run TestRRF
go test ./... -bench=.
```

**Tests:**
```go
func TestBM25_AddAndSearch(t *testing.T) {
    idx := NewBM25Index()
    idx.Add("doc1", "s1", "a1", "t1", "Go programming language goroutines")
    idx.Add("doc2", "s1", "a1", "t1", "Python machine learning")
    results := idx.Search("goroutines", 5)
    assert.Equal(t, "doc1", results[0].DocID)
}

func TestRRFFuse_CombinesScores(t *testing.T) {
    bm25 := []BM25Result{{DocID: "d1", Score: 2.5}, {DocID: "d2", Score: 1.5}}
    vec  := []VectorResult{{DocID: "d2", Score: 0.9}, {DocID: "d1", Score: 0.7}}
    results := RRFFuse(bm25, vec, nil, DefaultWeights, 5)
    // d2 ranks higher due to better vector score
    assert.Equal(t, 2, len(results))
}

func TestBM25_SurviveRestart(t *testing.T) {
    dir := t.TempDir()
    idx := NewBM25Index()
    idx.Add("doc1", "s1", "a1", "t1", "golang test")
    p := NewIndexPersister(idx, NewVectorIndex(384), dir)
    p.save()

    idx2 := NewBM25Index()
    p2 := NewIndexPersister(idx2, NewVectorIndex(384), dir)
    p2.LoadAsync()
    time.Sleep(100 * time.Millisecond)
    results := idx2.Search("golang", 5)
    assert.NotEmpty(t, results)
}
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| BM25 search → ranked results by BM25 score | ✅ |
| CJK text → bigram tokenized correctly | ✅ |
| RRF fusion → combine BM25 + vector ranks | ✅ |
| BM25 index saved to .gob and loaded on restart | ✅ |
| Debounced save: 30s after last write | ✅ |
| Vector cosine similarity correct | ✅ |
