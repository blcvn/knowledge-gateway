package analytics

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultAnalyticsTTL = 15 * time.Minute

type Cache struct {
	redis *redis.Client
	ttl   time.Duration
	memMu sync.RWMutex
	mem   map[string]cacheValue
}

func NewCache(redisCli *redis.Client) *Cache {
	return &Cache{
		redis: redisCli,
		ttl:   defaultAnalyticsTTL,
		mem:   make(map[string]cacheValue),
	}
}

type cacheValue struct {
	payload   []byte
	expiresAt time.Time
}

func (c *Cache) Get(ctx context.Context, reportType, namespace string, params any, out any) (bool, error) {
	if c == nil || c.redis == nil {
		return c.getMemory(reportType, namespace, params, out)
	}
	key, err := c.buildKey(reportType, namespace, params)
	if err != nil {
		return false, err
	}
	raw, err := c.redis.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	if raw == "" {
		return false, nil
	}
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		return false, err
	}
	return true, nil
}

func (c *Cache) Set(ctx context.Context, reportType, namespace string, params any, value any) error {
	if c == nil || c.redis == nil {
		return c.setMemory(reportType, namespace, params, value)
	}
	key, err := c.buildKey(reportType, namespace, params)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, key, payload, c.ttl).Err()
}

func (c *Cache) buildKey(reportType, namespace string, params any) (string, error) {
	payload, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	hasher := sha1.New() //nolint:gosec // non-cryptographic cache-key hashing is sufficient here.
	_, _ = hasher.Write(payload)
	hash := hex.EncodeToString(hasher.Sum(nil))
	return fmt.Sprintf("kgs:analytics:%s:%s:%s", reportType, namespace, hash), nil
}

func (c *Cache) getMemory(reportType, namespace string, params any, out any) (bool, error) {
	key, err := c.buildKey(reportType, namespace, params)
	if err != nil {
		return false, err
	}
	c.memMu.RLock()
	entry, ok := c.mem[key]
	c.memMu.RUnlock()
	if !ok {
		return false, nil
	}
	if time.Now().After(entry.expiresAt) {
		c.memMu.Lock()
		delete(c.mem, key)
		c.memMu.Unlock()
		return false, nil
	}
	if err := json.Unmarshal(entry.payload, out); err != nil {
		return false, err
	}
	return true, nil
}

func (c *Cache) setMemory(reportType, namespace string, params any, value any) error {
	key, err := c.buildKey(reportType, namespace, params)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.memMu.Lock()
	c.mem[key] = cacheValue{
		payload:   payload,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.memMu.Unlock()
	return nil
}
