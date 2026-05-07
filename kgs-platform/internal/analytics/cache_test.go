package analytics

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCacheSetGetWithMemoryFallback(t *testing.T) {
	cache := NewCache(nil)
	cache.ttl = 50 * time.Millisecond

	type payload struct {
		Value int `json:"value"`
	}

	if err := cache.Set(context.Background(), "coverage", "graph/app-1/tenant-1", map[string]any{"domain": "payment"}, payload{Value: 42}); err != nil {
		t.Fatalf("Set error: %v", err)
	}

	var out payload
	hit, err := cache.Get(context.Background(), "coverage", "graph/app-1/tenant-1", map[string]any{"domain": "payment"}, &out)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if !hit || out.Value != 42 {
		t.Fatalf("unexpected cache get hit=%v out=%#v", hit, out)
	}

	time.Sleep(60 * time.Millisecond)
	hit, err = cache.Get(context.Background(), "coverage", "graph/app-1/tenant-1", map[string]any{"domain": "payment"}, &out)
	if err != nil {
		t.Fatalf("Get after ttl error: %v", err)
	}
	if hit {
		t.Fatalf("expected expired cache miss")
	}
}

func TestCacheBuildKeyPattern(t *testing.T) {
	cache := NewCache(nil)
	keyA, err := cache.buildKey("coverage", "graph/app-1/tenant-1", map[string]any{"domain": "payment"})
	if err != nil {
		t.Fatalf("buildKey error: %v", err)
	}
	keyB, err := cache.buildKey("coverage", "graph/app-1/tenant-1", map[string]any{"domain": "order"})
	if err != nil {
		t.Fatalf("buildKey error: %v", err)
	}
	if !strings.HasPrefix(keyA, "kgs:analytics:coverage:graph/app-1/tenant-1:") {
		t.Fatalf("unexpected key pattern: %s", keyA)
	}
	if keyA == keyB {
		t.Fatalf("expected key hash to vary with params")
	}
}
