package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/domain/model"
	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/usecase/port"
)

type profileCache struct {
	client *redis.Client
}

func NewProfileCache(client *redis.Client) port.ProfileCache {
	return &profileCache{client: client}
}

func formatKey(projectID, userID string) string {
	return fmt.Sprintf("user_profiles::%s::%s", projectID, userID)
}

func (c *profileCache) GetProfiles(ctx context.Context, userID, projectID string) ([]*model.Profile, error) {
	val, err := c.client.Get(ctx, formatKey(projectID, userID)).Result()
	if err != nil {
		return nil, err
	}
	var profiles []*model.Profile
	if err := json.Unmarshal([]byte(val), &profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (c *profileCache) SetProfiles(ctx context.Context, userID, projectID string, profiles []*model.Profile, ttlSeconds int) error {
	data, err := json.Marshal(profiles)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, formatKey(projectID, userID), data, time.Duration(ttlSeconds)*time.Second).Err()
}

func (c *profileCache) DeleteProfiles(ctx context.Context, userID, projectID string) error {
	return c.client.Del(ctx, formatKey(projectID, userID)).Err()
}
