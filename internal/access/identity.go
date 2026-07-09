package access

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"kg-service/internal/platform/rediscache"
)

const identityCacheTTL = 30 * time.Second

var ErrUnauthorized = errors.New("unauthorized")

type IdentityResolver struct {
	apps  AppStore
	cache *rediscache.Client
}

type identityCacheEntry struct {
	TenantID string `json:"tenant_id"`
	AppID    string `json:"app_id"`
	AppType  string `json:"app_type"`
	Status   string `json:"status"`
}

func NewIdentityResolver(apps AppStore, cache *rediscache.Client) *IdentityResolver {
	return &IdentityResolver{apps: apps, cache: cache}
}

func (r *IdentityResolver) Resolve(authorizationHeader string) (Identity, error) {
	if r == nil || r.apps == nil {
		return Identity{}, ErrUnauthorized
	}
	apiKey, err := bearerToken(authorizationHeader)
	if err != nil {
		return Identity{}, err
	}

	hash := APIKeyHash(apiKey)
	cacheKey := "apikey:" + hash

	var cached identityCacheEntry
	cacheHit := false
	if r != nil && r.cache != nil {
		if ok, err := r.cache.GetJSON(cacheKey, &cached); err == nil && ok {
			cacheHit = true
		}
	}

	app, ok := r.apps.GetAppByAPIKeyHash(hash)
	if !ok {
		if cacheHit && cached.Status == "active" {
			return Identity{TenantID: cached.TenantID, AppID: cached.AppID, AppType: cached.AppType}, nil
		}
		if r != nil && r.cache != nil {
			r.cache.Delete(cacheKey)
		}
		return Identity{}, ErrUnauthorized
	}
	if app.Status != "active" {
		if r != nil && r.cache != nil {
			r.cache.Delete(cacheKey)
		}
		return Identity{}, ErrUnauthorized
	}

	entry := identityCacheEntry{
		TenantID: app.TenantID,
		AppID:    app.ID,
		AppType:  app.Type,
		Status:   app.Status,
	}
	if cacheHit && cached == entry {
		return Identity{TenantID: app.TenantID, AppID: app.ID, AppType: app.Type}, nil
	}
	if r != nil && r.cache != nil {
		if err := r.cache.SetJSON(cacheKey, entry, identityCacheTTL); err != nil {
			return Identity{}, err
		}
	}

	return Identity{TenantID: app.TenantID, AppID: app.ID, AppType: app.Type}, nil
}

func bearerToken(header string) (string, error) {
	if header == "" {
		return "", ErrUnauthorized
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", fmt.Errorf("%w: invalid authorization scheme", ErrUnauthorized)
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", ErrUnauthorized
	}

	return token, nil
}
