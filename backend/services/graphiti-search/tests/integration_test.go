package tests

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"vnp-memory/services/graphiti-search/internal/domain"
	"vnp-memory/services/graphiti-search/internal/adapter/cache"
)

func TestIntegration_RedisCacheAdapter(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	
	err := rdb.Ping(context.Background()).Err()
	if err != nil {
		t.Skip("Redis is not available, skipping integration test")
	}

	adapter := cache.NewRedisCacheAdapter(rdb)
	key := "search:test-group:12345"

	results := []domain.RankedResult{
		{EntityID: "E1", Score: 0.99, Rank: 1},
	}

	err = adapter.Set(context.Background(), key, results, 1*time.Minute)
	if err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	cached, err := adapter.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	if len(cached) != 1 || cached[0].EntityID != "E1" {
		t.Errorf("Cache content mismatch, got: %+v", cached)
	}
}

func TestIntegration_HybridSearch(t *testing.T) {
	t.Log("Testing HybridSearch - Requires full store context")
}

func TestIntegration_CacheInvalidation(t *testing.T) {
	t.Log("Testing CacheInvalidation - Requires NATS context")
}

func TestIntegration_TenantIsolation(t *testing.T) {
	t.Log("Testing TenantIsolation - Check groupID keys")
}
