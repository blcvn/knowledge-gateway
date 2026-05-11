package redis

import (
	"context"
	"time"

	"vnp-memory/services/cognee-search/internal/domain"
	"vnp-memory/services/cognee-search/internal/usecase/port"
)

type cacheStore struct {
	ttl time.Duration
}

func NewCacheStore(ttl time.Duration) port.CacheStore {
	return &cacheStore{
		ttl: ttl,
	}
}

func (s *cacheStore) Get(ctx context.Context, key string) ([]domain.SearchResult, error) {
	return nil, nil // cache miss
}

func (s *cacheStore) Set(ctx context.Context, key string, results []domain.SearchResult) error {
	return nil
}

func (s *cacheStore) Invalidate(ctx context.Context, pattern string) error {
	return nil
}
