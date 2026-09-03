package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"vnp-memory/services/graphiti-search/internal/domain"
	"vnp-memory/services/graphiti-search/internal/usecase"
)

type RedisCacheAdapter struct {
	rdb *redis.Client
}

func NewRedisCacheAdapter(rdb *redis.Client) usecase.CacheRepo {
	return &RedisCacheAdapter{rdb: rdb}
}

func (r *RedisCacheAdapter) Get(ctx context.Context, key string) ([]domain.RankedResult, error) {
	val, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, domain.ErrNoResults
		}
		return nil, domain.ErrCacheUnavailable
	}

	var results []domain.RankedResult
	if err := json.Unmarshal([]byte(val), &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *RedisCacheAdapter) Set(ctx context.Context, key string, results []domain.RankedResult, ttl time.Duration) error {
	data, err := json.Marshal(results)
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, key, data, ttl).Err()
}

func (r *RedisCacheAdapter) InvalidateGroup(ctx context.Context, groupID string) error {
	pattern := "search:" + groupID + ":*"
	keys, err := r.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return r.rdb.Del(ctx, keys...).Err()
	}
	return nil
}
