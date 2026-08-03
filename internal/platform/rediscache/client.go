package rediscache

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"kg-service/internal/config"
)

type Client struct {
	Address  string
	Username string
	Password string
	DB       int

	mu      sync.RWMutex
	entries map[string]entry
}

type entry struct {
	value     []byte
	expiresAt time.Time
}

func New(cfg config.RedisConfig) (Client, error) {
	if cfg.Address() == "" {
		return Client{}, fmt.Errorf("redis address must not be empty")
	}

	return Client{
		Address:  cfg.Address(),
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
		entries:  make(map[string]entry),
	}, nil
}

func (c *Client) SetJSON(key string, value any, ttl time.Duration) error {
	if c == nil {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = entry{
		value:     payload,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

func (c *Client) GetJSON(key string, target any) (bool, error) {
	if c == nil {
		return false, nil
	}
	c.mu.RLock()
	item, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return false, nil
	}

	if time.Now().After(item.expiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return false, nil
	}

	if err := json.Unmarshal(item.value, target); err != nil {
		return false, err
	}

	return true, nil
}

func (c *Client) Delete(key string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

func (c *Client) DeletePrefix(prefix string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.entries {
		if strings.HasPrefix(key, prefix) {
			delete(c.entries, key)
		}
	}
}
