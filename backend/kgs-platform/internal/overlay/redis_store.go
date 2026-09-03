package overlay

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	overlayKeyPrefix        = "kgs:overlay:"
	overlaySessionKeyPrefix = "kgs:overlay:session:"
)

type Store interface {
	Save(ctx context.Context, overlay *OverlayGraph, ttl time.Duration) error
	Get(ctx context.Context, overlayID string) (*OverlayGraph, error)
	Delete(ctx context.Context, overlayID string) error
	BindSession(ctx context.Context, sessionID, overlayID string, ttl time.Duration) error
	UnbindSession(ctx context.Context, sessionID string) error
	FindBySession(ctx context.Context, sessionID string) (string, error)
}

type RedisStore struct {
	redis *redis.Client
}

func NewRedisStore(redisClient *redis.Client) *RedisStore {
	return &RedisStore{redis: redisClient}
}

func (s *RedisStore) Save(ctx context.Context, overlay *OverlayGraph, ttl time.Duration) error {
	if s == nil || s.redis == nil || overlay == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = time.Until(overlay.ExpiresAt)
		if ttl <= 0 {
			ttl = time.Hour
		}
	}
	buf, err := json.Marshal(overlay)
	if err != nil {
		return err
	}
	return s.redis.Set(ctx, overlayKeyPrefix+overlay.OverlayID, string(buf), ttl).Err()
}

func (s *RedisStore) Get(ctx context.Context, overlayID string) (*OverlayGraph, error) {
	if s == nil || s.redis == nil {
		return nil, fmt.Errorf("overlay store unavailable")
	}
	raw, err := s.redis.Get(ctx, overlayKeyPrefix+overlayID).Result()
	if err != nil {
		return nil, err
	}
	var out OverlayGraph
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *RedisStore) Delete(ctx context.Context, overlayID string) error {
	if s == nil || s.redis == nil {
		return nil
	}
	return s.redis.Del(ctx, overlayKeyPrefix+overlayID).Err()
}

func (s *RedisStore) BindSession(ctx context.Context, sessionID, overlayID string, ttl time.Duration) error {
	if s == nil || s.redis == nil || sessionID == "" || overlayID == "" {
		return nil
	}
	if ttl <= 0 {
		ttl = time.Hour
	}
	return s.redis.Set(ctx, overlaySessionKeyPrefix+sessionID, overlayID, ttl).Err()
}

func (s *RedisStore) UnbindSession(ctx context.Context, sessionID string) error {
	if s == nil || s.redis == nil || sessionID == "" {
		return nil
	}
	return s.redis.Del(ctx, overlaySessionKeyPrefix+sessionID).Err()
}

func (s *RedisStore) FindBySession(ctx context.Context, sessionID string) (string, error) {
	if s == nil || s.redis == nil || sessionID == "" {
		return "", nil
	}
	value, err := s.redis.Get(ctx, overlaySessionKeyPrefix+sessionID).Result()
	if err == redis.Nil {
		return "", nil
	}
	return value, err
}

var _ Store = (*RedisStore)(nil)
