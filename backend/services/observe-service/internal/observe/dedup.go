package observe

import (
    "context"
    "sync"
    "time"
)

const DefaultDedupTTL = 30 * time.Second

type DedupMap struct {
    mu    sync.RWMutex
    store map[[32]byte]time.Time
}

func NewDedupMap() *DedupMap {
    return &DedupMap{store: make(map[[32]byte]time.Time)}
}

func (d *DedupMap) IsSeen(hash [32]byte) bool {
    d.mu.RLock()
    defer d.mu.RUnlock()
    exp, ok := d.store[hash]
    return ok && time.Now().Before(exp)
}

func (d *DedupMap) MarkSeen(hash [32]byte, ttl time.Duration) {
    d.mu.Lock()
    defer d.mu.Unlock()
    d.store[hash] = time.Now().Add(ttl)
}

// StartCleanup runs every 60s to clear expired entries
func (d *DedupMap) StartCleanup(ctx context.Context) {
    ticker := time.NewTicker(60 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            d.mu.Lock()
            now := time.Now()
            for hash, exp := range d.store {
                if now.After(exp) { delete(d.store, hash) }
            }
            d.mu.Unlock()
        case <-ctx.Done():
            return
        }
    }
}
