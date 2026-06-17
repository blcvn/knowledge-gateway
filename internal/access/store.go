package access

import (
	"slices"
	"sync"
	"time"
)

type TenantStore interface {
	GetTenant(id string) (Tenant, bool)
	CreateTenant(tenant Tenant) Tenant
	UpdateTenant(tenant Tenant) (Tenant, bool)
}

type AppStore interface {
	GetAppByAPIKeyHash(hash string) (App, bool)
	GetAppByID(id string) (App, bool)
	ListAppsByTenant(tenantID string) []App
	CreateApp(app App) App
	UpdateApp(app App) (App, bool)
}

type GrantStore interface {
	ListGrantsForGrantee(tenantID, appID string) []AccessGrant
	CreateGrant(grant AccessGrant) AccessGrant
	ListGrants(filter GrantListFilter) []AccessGrant
	GetGrantByID(id string) (AccessGrant, bool)
	UpdateGrant(grant AccessGrant) (AccessGrant, bool)
	CreateAuditLog(entry AuditLogEntry) AuditLogEntry
	ListAuditLogs(filter AuditListFilter) []AuditLogEntry
}

type MemoryStore struct {
	mu      sync.RWMutex
	tenants map[string]Tenant
	apps    map[string]App
	appIDs  map[string]string
	grants  []AccessGrant
	audit   []AuditLogEntry
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tenants: map[string]Tenant{},
		apps:    map[string]App{},
		appIDs:  map[string]string{},
		grants:  []AccessGrant{},
		audit:   []AuditLogEntry{},
	}
}

func (s *MemoryStore) Seed(tenants []Tenant, apps []App, grants []AccessGrant) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, tenant := range tenants {
		s.tenants[tenant.ID] = tenant
	}
	for _, app := range apps {
		s.apps[app.APIKeyHash] = app
		s.appIDs[app.ID] = app.APIKeyHash
	}
	s.grants = append([]AccessGrant(nil), grants...)
}

func (s *MemoryStore) GetTenant(id string) (Tenant, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tenant, ok := s.tenants[id]
	return tenant, ok
}

func (s *MemoryStore) GetAppByAPIKeyHash(hash string) (App, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	app, ok := s.apps[hash]
	return app, ok
}

func (s *MemoryStore) GetAppByID(id string) (App, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hash, ok := s.appIDs[id]
	if !ok {
		return App{}, false
	}
	app, ok := s.apps[hash]
	return app, ok
}

func (s *MemoryStore) ListAppsByTenant(tenantID string) []App {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var apps []App
	for _, app := range s.apps {
		if app.TenantID == tenantID {
			apps = append(apps, app)
		}
	}
	slices.SortFunc(apps, func(a, b App) int {
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		if a.CreatedAt.After(b.CreatedAt) {
			return 1
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return apps
}

func (s *MemoryStore) CreateTenant(tenant Tenant) Tenant {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenants[tenant.ID] = tenant
	return tenant
}

func (s *MemoryStore) UpdateTenant(tenant Tenant) (Tenant, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tenants[tenant.ID]; !ok {
		return Tenant{}, false
	}
	tenant.UpdatedAt = time.Now().UTC()
	s.tenants[tenant.ID] = tenant
	return tenant, true
}

func (s *MemoryStore) CreateApp(app App) App {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.apps[app.APIKeyHash] = app
	s.appIDs[app.ID] = app.APIKeyHash
	return app
}

func (s *MemoryStore) UpdateApp(app App) (App, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	oldHash, ok := s.appIDs[app.ID]
	if !ok {
		return App{}, false
	}
	delete(s.apps, oldHash)
	s.apps[app.APIKeyHash] = app
	s.appIDs[app.ID] = app.APIKeyHash
	return app, true
}

func (s *MemoryStore) ListGrantsForGrantee(tenantID, appID string) []AccessGrant {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var grants []AccessGrant
	for _, grant := range s.grants {
		if grant.GranteeTenantID != tenantID {
			continue
		}
		if grant.GranteeAppID != "" && grant.GranteeAppID != appID {
			continue
		}
		grants = append(grants, grant)
	}

	return grants
}

func (s *MemoryStore) CreateGrant(grant AccessGrant) AccessGrant {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.grants = append(s.grants, grant)
	return grant
}

func (s *MemoryStore) ListGrants(filter GrantListFilter) []AccessGrant {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var grants []AccessGrant
	for _, grant := range s.grants {
		if filter.GrantorTenantID != "" && grant.GrantorTenantID != filter.GrantorTenantID {
			continue
		}
		if filter.GranteeTenantID != "" && grant.GranteeTenantID != filter.GranteeTenantID {
			continue
		}
		grants = append(grants, grant)
	}
	slices.SortFunc(grants, func(a, b AccessGrant) int {
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		if a.CreatedAt.After(b.CreatedAt) {
			return 1
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return grants
}

func (s *MemoryStore) GetGrantByID(id string) (AccessGrant, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, grant := range s.grants {
		if grant.ID == id {
			return grant, true
		}
	}
	return AccessGrant{}, false
}

func (s *MemoryStore) UpdateGrant(grant AccessGrant) (AccessGrant, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.grants {
		if s.grants[i].ID == grant.ID {
			s.grants[i] = grant
			return grant, true
		}
	}
	return AccessGrant{}, false
}

func (s *MemoryStore) CreateAuditLog(entry AuditLogEntry) AuditLogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audit = append(s.audit, entry)
	return entry
}

func (s *MemoryStore) ListAuditLogs(filter AuditListFilter) []AuditLogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var entries []AuditLogEntry
	for _, entry := range s.audit {
		if filter.ResourceOwnerTenantID != "" && entry.ResourceOwnerTenantID != filter.ResourceOwnerTenantID {
			continue
		}
		entries = append(entries, entry)
	}
	slices.SortFunc(entries, func(a, b AuditLogEntry) int {
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		if a.CreatedAt.After(b.CreatedAt) {
			return 1
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return paginateAudit(entries, filter.Limit, filter.Cursor)
}

func paginateAudit(entries []AuditLogEntry, limit int, cursor string) []AuditLogEntry {
	return paginateAuditByID(entries, limit, cursor)
}

func paginateAuditByID(entries []AuditLogEntry, limit int, cursor string) []AuditLogEntry {
	if limit <= 0 || limit > len(entries) {
		limit = len(entries)
	}
	start := 0
	if cursor != "" {
		for i, item := range entries {
			if item.ID == cursor {
				start = i + 1
				break
			}
		}
	}
	end := start + limit
	if end > len(entries) {
		end = len(entries)
	}
	if start > len(entries) {
		return []AuditLogEntry{}
	}
	return append([]AuditLogEntry(nil), entries[start:end]...)
}
