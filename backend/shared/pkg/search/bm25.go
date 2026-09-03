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
