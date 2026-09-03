package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// LLMCache — interface for caching LLM responses
type LLMCache interface {
	Get(ctx context.Context, key string) (*CachedResponse, bool)
	Set(ctx context.Context, key string, content []byte, ttl time.Duration)
}

type CachedResponse struct {
	Content []byte `json:"content"`
}

// RedisLLMCache caches LLM responses by message hash (1h TTL default)
type RedisLLMCache struct {
	client redis.UniversalClient
	prefix string
}

func NewRedisLLMCache(client redis.UniversalClient) *RedisLLMCache {
	return &RedisLLMCache{client: client, prefix: "graphiti:llm:"}
}

func (c *RedisLLMCache) Get(ctx context.Context, key string) (*CachedResponse, bool) {
	val, err := c.client.Get(ctx, c.prefix+key).Bytes()
	if err != nil {
		return nil, false
	}
	var cached CachedResponse
	if err := json.Unmarshal(val, &cached); err != nil {
		return nil, false
	}
	return &cached, true
}

func (c *RedisLLMCache) Set(ctx context.Context, key string, content []byte, ttl time.Duration) {
	data, err := json.Marshal(CachedResponse{Content: content})
	if err != nil {
		return
	}
	c.client.Set(ctx, c.prefix+key, data, ttl)
}

// NoopLLMCache — passthrough cache (no caching). Used in tests / dev.
type NoopLLMCache struct{}

func (n *NoopLLMCache) Get(ctx context.Context, key string) (*CachedResponse, bool) {
	return nil, false
}
func (n *NoopLLMCache) Set(ctx context.Context, key string, content []byte, ttl time.Duration) {}
