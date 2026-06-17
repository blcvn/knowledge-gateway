package access

import (
	"slices"
	"time"

	"kg-service/internal/platform/rediscache"
)

const accessCacheTTL = 60 * time.Second

type AccessResolver struct {
	tenants TenantStore
	grants  GrantStore
	cache   *rediscache.Client
}

func NewAccessResolver(tenants TenantStore, grants GrantStore, cache *rediscache.Client) *AccessResolver {
	return &AccessResolver{tenants: tenants, grants: grants, cache: cache}
}

func (r *AccessResolver) ResolveVisibleOwners(identity Identity) ([]VisibleOwner, error) {
	cacheKey := "acl:" + identity.TenantID + ":" + identity.AppID

	var cached []VisibleOwner
	if ok, err := r.cache.GetJSON(cacheKey, &cached); err == nil && ok {
		return cached, nil
	}

	owners := []VisibleOwner{
		{TenantID: identity.TenantID, AppID: identity.AppID, Permission: "admin", Source: "self"},
		{TenantID: identity.TenantID, Permission: "read", Source: "tenant"},
		{TenantID: PlatformTenantID, Permission: "read", Source: "platform"},
	}

	if tenant, ok := r.tenants.GetTenant(identity.TenantID); ok && tenant.DefaultSharingPolicy == "share_within_tenant_read" {
		owners = append(owners, VisibleOwner{
			TenantID:   identity.TenantID,
			AppID:      "*",
			Permission: "read",
			Source:     "tenant_policy",
		})
	}

	now := time.Now()
	for _, grant := range r.grants.ListGrantsForGrantee(identity.TenantID, identity.AppID) {
		if grant.Status != "active" {
			continue
		}
		if grant.ExpiresAt != nil && !grant.ExpiresAt.After(now) {
			continue
		}

		owners = append(owners, VisibleOwner{
			TenantID:   grant.GrantorTenantID,
			AppID:      fallbackWildcard(grant.GrantorAppID),
			Permission: grant.Permission,
			ScopeType:  grant.ScopeType,
			ScopeValue: grant.ScopeValue,
			Source:     "grant",
		})
	}

	owners = dedupeVisibleOwners(owners)
	if err := r.cache.SetJSON(cacheKey, owners, accessCacheTTL); err != nil {
		return nil, err
	}

	return owners, nil
}

func fallbackWildcard(value string) string {
	if value == "" {
		return "*"
	}
	return value
}

func dedupeVisibleOwners(owners []VisibleOwner) []VisibleOwner {
	seen := map[VisibleOwner]struct{}{}
	result := make([]VisibleOwner, 0, len(owners))
	for _, owner := range owners {
		if _, ok := seen[owner]; ok {
			continue
		}
		seen[owner] = struct{}{}
		result = append(result, owner)
	}

	slices.SortFunc(result, func(a, b VisibleOwner) int {
		if a.Source != b.Source {
			if a.Source < b.Source {
				return -1
			}
			return 1
		}
		if a.TenantID != b.TenantID {
			if a.TenantID < b.TenantID {
				return -1
			}
			return 1
		}
		if a.AppID != b.AppID {
			if a.AppID < b.AppID {
				return -1
			}
			return 1
		}
		return 0
	})

	return result
}
