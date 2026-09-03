// Package persistence provides infrastructure-layer data store implementations.
package persistence

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter implements port.RateLimitStore using Redis sorted sets (sliding window).
type RedisRateLimiter struct {
	client *redis.Client
	logger *slog.Logger
}

// NewRedisRateLimiter creates a new Redis-backed rate limiter.
func NewRedisRateLimiter(client *redis.Client, logger *slog.Logger) *RedisRateLimiter {
	return &RedisRateLimiter{client: client, logger: logger}
}

// slidingWindowScript is the atomic Lua script for sliding window rate limiting.
// Uses Redis sorted sets: each member is a timestamp, score is the same timestamp.
// Steps:
//  1. Remove entries outside the window
//  2. Count remaining entries
//  3. If under limit, add new entry
//  4. Set expiry on the key
//
// Returns: [allowed (0/1), remaining count]
var slidingWindowScript = redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local member = ARGV[4]

-- Remove expired entries outside the sliding window
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

-- Count current entries in the window
local count = redis.call('ZCARD', key)

if count < limit then
    -- Under limit: add entry and return allowed
    redis.call('ZADD', key, now, member)
    redis.call('EXPIRE', key, window)
    return {1, limit - count - 1}
else
    -- Over limit: reject
    redis.call('EXPIRE', key, window)
    return {0, 0}
end
`)

// CheckAndIncrement atomically checks and increments the sliding window counter.
// Returns whether the request is allowed and the remaining quota.
func (r *RedisRateLimiter) CheckAndIncrement(ctx context.Context, key string, limit int, windowSec int) (bool, int, error) {
	now := time.Now().UnixMilli()
	windowMs := int64(windowSec) * 1000
	member := fmt.Sprintf("%d:%d", now, time.Now().UnixNano()%1000000)

	result, err := slidingWindowScript.Run(ctx, r.client,
		[]string{key},
		now,
		windowMs,
		limit,
		member,
	).Int64Slice()

	if err != nil {
		r.logger.Error("redis rate limit script failed",
			"key", key,
			"error", err,
		)
		return false, 0, fmt.Errorf("rate limit check: %w", err)
	}

	allowed := result[0] == 1
	remaining := int(result[1])

	if !allowed {
		r.logger.Debug("rate limit rejected", "key", key, "limit", limit)
	}

	return allowed, remaining, nil
}

// NewRedisClient creates a new Redis client with the given address.
func NewRedisClient(addr, password string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     20,
		MinIdleConns: 5,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return client, nil
}
