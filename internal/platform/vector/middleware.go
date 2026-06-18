package vector

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"
)

type CachingProvider struct {
	Inner   EmbeddingProvider
	TTL     time.Duration
	Now     func() time.Time
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

type cacheEntry struct {
	value     []float64
	expiresAt time.Time
}

func (p *CachingProvider) Dimensions() int {
	if inner := p.inner(); inner != nil {
		return inner.Dimensions()
	}
	return 0
}

func (p *CachingProvider) ModelID() string {
	if inner := p.inner(); inner != nil {
		return inner.ModelID()
	}
	return ""
}

func (p *CachingProvider) Embed(ctx context.Context, text string) ([]float64, error) {
	inner := p.inner()
	if inner == nil {
		return nil, errors.New("caching provider requires an inner provider")
	}
	now := p.now()
	key := cacheKey(inner.ModelID(), text)
	p.mu.RLock()
	entry, ok := p.entries[key]
	p.mu.RUnlock()
	if ok && (entry.expiresAt.IsZero() || now.Before(entry.expiresAt)) {
		return append([]float64(nil), entry.value...), nil
	}
	vec, err := inner.Embed(ctx, text)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	if p.entries == nil {
		p.entries = map[string]cacheEntry{}
	}
	expiresAt := time.Time{}
	if p.TTL > 0 {
		expiresAt = now.Add(p.TTL)
	}
	p.entries[key] = cacheEntry{value: append([]float64(nil), vec...), expiresAt: expiresAt}
	p.mu.Unlock()
	return vec, nil
}

func (p *CachingProvider) inner() EmbeddingProvider {
	if p == nil {
		return nil
	}
	return p.Inner
}

func (p *CachingProvider) now() time.Time {
	if p != nil && p.Now != nil {
		return p.Now()
	}
	return time.Now().UTC()
}

type RetryProvider struct {
	Inner        EmbeddingProvider
	MaxAttempts  int
	InitialDelay time.Duration
	Sleep        func(time.Duration)
}

func (p *RetryProvider) Dimensions() int {
	if inner := p.inner(); inner != nil {
		return inner.Dimensions()
	}
	return 0
}

func (p *RetryProvider) ModelID() string {
	if inner := p.inner(); inner != nil {
		return inner.ModelID()
	}
	return ""
}

func (p *RetryProvider) Embed(ctx context.Context, text string) ([]float64, error) {
	inner := p.inner()
	if inner == nil {
		return nil, errors.New("retry provider requires an inner provider")
	}
	attempts := p.MaxAttempts
	if attempts <= 0 {
		attempts = 3
	}
	delay := p.InitialDelay
	if delay <= 0 {
		delay = 25 * time.Millisecond
	}
	sleep := p.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		vec, err := inner.Embed(ctx, text)
		if err == nil {
			return vec, nil
		}
		lastErr = err
		if attempt < attempts {
			sleep(delay)
			delay *= 2
		}
	}
	return nil, lastErr
}

func (p *RetryProvider) inner() EmbeddingProvider {
	if p == nil {
		return nil
	}
	return p.Inner
}

type ProxyHTTPProvider struct {
	Inner    EmbeddingProvider
	ProxyURL string
}

func (p ProxyHTTPProvider) Dimensions() int {
	if p.Inner == nil {
		return 0
	}
	return p.Inner.Dimensions()
}

func (p ProxyHTTPProvider) ModelID() string {
	if p.Inner == nil {
		return ""
	}
	return p.Inner.ModelID()
}

func (p ProxyHTTPProvider) Embed(ctx context.Context, text string) ([]float64, error) {
	if p.Inner == nil {
		return nil, errors.New("proxy provider requires an inner provider")
	}
	if strings.TrimSpace(p.ProxyURL) == "" {
		return p.Inner.Embed(ctx, text)
	}
	switch inner := p.Inner.(type) {
	case HTTPEmbeddingProvider:
		clone := inner
		clone.URL = p.ProxyURL
		return clone.Embed(ctx, text)
	case *HTTPEmbeddingProvider:
		if inner == nil {
			return nil, errors.New("proxy provider requires an inner provider")
		}
		clone := *inner
		clone.URL = p.ProxyURL
		return clone.Embed(ctx, text)
	default:
		return p.Inner.Embed(ctx, text)
	}
}

func cacheKey(modelID, text string) string {
	sum := sha256.Sum256([]byte(modelID + "\n" + text))
	return hex.EncodeToString(sum[:])
}
