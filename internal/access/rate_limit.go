package access

import (
	"strings"
	"sync"
	"time"
)

type Limiter interface {
	Allow(identity Identity) bool
}

type RateLimiter struct {
	store   TenantStore
	now     func() time.Time
	mu      sync.Mutex
	limits  map[string]int
	buckets map[string]tierBucket
}

type tierBucket struct {
	windowStart time.Time
	count       int
}

func NewRateLimiter(store TenantStore, tierLimits ...map[string]int) *RateLimiter {
	limits := map[string]int{
		"free":       15,
		"pro":        60,
		"enterprise": 240,
	}
	if len(tierLimits) > 0 && tierLimits[0] != nil {
		for tier, limit := range tierLimits[0] {
			limits[strings.ToLower(strings.TrimSpace(tier))] = limit
		}
	}
	return &RateLimiter{
		store:   store,
		now:     func() time.Time { return time.Now().UTC() },
		limits:  limits,
		buckets: map[string]tierBucket{},
	}
}

func (l *RateLimiter) SetNow(now func() time.Time) {
	if l == nil || now == nil {
		return
	}
	l.now = now
}

func (l *RateLimiter) Allow(identity Identity) bool {
	if l == nil || l.store == nil {
		return true
	}
	tenant, ok := l.store.GetTenant(identity.TenantID)
	if !ok {
		return false
	}
	tier := strings.ToLower(strings.TrimSpace(tenant.Tier))
	if tier == "" {
		tier = "free"
	}
	limit, ok := l.limits[tier]
	if !ok || limit <= 0 {
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now().UTC().Truncate(time.Minute)
	key := identity.TenantID + ":" + tier
	bucket := l.buckets[key]
	if bucket.windowStart.IsZero() || !bucket.windowStart.Equal(now) {
		bucket.windowStart = now
		bucket.count = 0
	}
	if bucket.count >= limit {
		l.buckets[key] = bucket
		return false
	}
	bucket.count++
	l.buckets[key] = bucket
	return true
}
