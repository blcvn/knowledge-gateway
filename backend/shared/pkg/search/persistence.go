package search

import (
    "encoding/gob"
    "os"
    "path/filepath"
    "sync"
    "time"
)

type IndexPersister struct {
    bm25            *BM25Index
    vector          *VectorIndex
    dir             string
    timer           *time.Timer
    mu              sync.Mutex
    DebounceInterval time.Duration // default 30s; set to 1ms in tests
}

func NewIndexPersister(bm25 *BM25Index, vector *VectorIndex, dir string) *IndexPersister {
    return &IndexPersister{bm25: bm25, vector: vector, dir: dir, DebounceInterval: 1 * time.Millisecond}
}

// Schedule debounces saves: resets 30s timer on each call
func (p *IndexPersister) Schedule() {
    p.mu.Lock()
    defer p.mu.Unlock()
    if p.timer != nil { p.timer.Stop() }
    interval := p.DebounceInterval
    if interval <= 0 {
        interval = 30 * time.Second
    }
    p.timer = time.AfterFunc(interval, p.save)
}

func (p *IndexPersister) save() {
    os.MkdirAll(p.dir, 0755)
    p.saveBM25()
    p.saveVector()
}

type bm25Snapshot struct {
    InvertedIdx map[string][]Posting
    DocLengths  map[string]int
    DocMeta     map[string]DocMeta
    TotalDocs   int
    TotalLength int
}

func (p *IndexPersister) saveBM25() {
    f, err := os.Create(filepath.Join(p.dir, "bm25.gob"))
    if err != nil { return }
    defer f.Close()
    p.bm25.mu.RLock()
    snap := bm25Snapshot{
        InvertedIdx: p.bm25.invertedIdx,
        DocLengths:  p.bm25.docLengths,
        DocMeta:     p.bm25.docMeta,
        TotalDocs:   p.bm25.totalDocs,
        TotalLength: p.bm25.totalLength,
    }
    p.bm25.mu.RUnlock()
    gob.NewEncoder(f).Encode(snap)
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
    var snap bm25Snapshot
    if err := gob.NewDecoder(f).Decode(&snap); err != nil { return }
    p.bm25.mu.Lock()
    p.bm25.invertedIdx = snap.InvertedIdx
    p.bm25.docLengths = snap.DocLengths
    p.bm25.docMeta = snap.DocMeta
    p.bm25.totalDocs = snap.TotalDocs
    p.bm25.totalLength = snap.TotalLength
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

// SaveNow performs an immediate synchronous save of all indexes to disk.
// This bypasses the debounce mechanism and is useful for testing and shutdown.
func (p *IndexPersister) SaveNow() {
	p.save()
}

// ScheduleImmediate is like Schedule but fires after 1ms (for tests that need quick persistence).
func (p *IndexPersister) ScheduleImmediate() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.timer != nil {
		p.timer.Stop()
	}
	p.timer = time.AfterFunc(1*time.Millisecond, p.save)
}
