package access

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kg-service/internal/config"
	"kg-service/internal/platform/rediscache"
)

func TestIdentityResolverResolvesAndCachesActiveApp(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), nil)

	resolver := NewIdentityResolver(store, cache)
	identity, err := resolver.Resolve("Bearer kgsk_test_alpha")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if identity.TenantID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("TenantID = %q", identity.TenantID)
	}
	if identity.AppID != "11111111-aaaa-1111-aaaa-111111111111" {
		t.Fatalf("AppID = %q", identity.AppID)
	}

	store = NewMemoryStore()
	resolver = NewIdentityResolver(store, cache)
	identity, err = resolver.Resolve("Bearer kgsk_test_alpha")
	if err != nil {
		t.Fatalf("Resolve() from cache error = %v", err)
	}
	if identity.AppID == "" {
		t.Fatal("cached identity is empty")
	}
}

func TestAccessResolverIncludesPlatformTenantAndGrants(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), SeedGrants())

	resolver := NewAccessResolver(store, store, cache)
	owners, err := resolver.ResolveVisibleOwners(Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-aaaa-1111-aaaa-111111111111",
	})
	if err != nil {
		t.Fatalf("ResolveVisibleOwners() error = %v", err)
	}

	if len(owners) < 4 {
		t.Fatalf("owners len = %d, want at least 4", len(owners))
	}

	foundPlatform := false
	foundGrant := false
	for _, owner := range owners {
		if owner.TenantID == PlatformTenantID {
			foundPlatform = true
		}
		if owner.Source == "grant" && owner.ScopeValue == "shared-domain" {
			foundGrant = true
		}
	}
	if !foundPlatform {
		t.Fatal("expected platform visibility owner")
	}
	if !foundGrant {
		t.Fatal("expected grant-derived visibility")
	}
}

func TestMiddlewareSanitizesIdentityFieldsAndInjectsContext(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), nil)

	middleware := NewMiddleware(NewIdentityResolver(store, cache))

	handler := middleware.RequireIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := IdentityFromContext(r.Context())
		if !ok {
			t.Fatal("identity missing from context")
		}
		if identity.AppID == "" {
			t.Fatal("identity app id missing")
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if strings.Contains(string(body), "tenant_id") || strings.Contains(string(body), "app_id") {
			t.Fatalf("sanitized body still contains identity fields: %s", string(body))
		}

		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodPost, "/v1/access/resolve", strings.NewReader(`{"tenant_id":"fake","app_id":"fake","foo":"bar"}`))
	request.Header.Set("Authorization", "Bearer kgsk_test_alpha")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestGetResolveReturnsIdentityVisibility(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), SeedGrants())

	handler := NewHandler(NewAccessResolver(store, store, cache), NewService(store, cache))
	request := httptest.NewRequest(http.MethodGet, "/v1/access/resolve", nil)
	request = request.WithContext(ContextWithIdentity(request.Context(), Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-aaaa-1111-aaaa-111111111111",
	}))
	recorder := httptest.NewRecorder()

	handler.GetResolve(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"visible_owners"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func TestServiceCreateTenantRequiresPlatformAdmin(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), nil)
	service := NewService(store, cache)

	_, err := service.CreateTenant(Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-aaaa-1111-aaaa-111111111111",
		AppType:  "agent_consumer",
	}, TenantCreateRequest{
		Slug: "acme",
		Name: "ACME",
		Tier: "pro",
	})
	if err == nil {
		t.Fatal("CreateTenant() error = nil, want forbidden")
	}
}

func TestServiceCreateAppListRotateAndRevoke(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), nil)
	service := NewService(store, cache)
	actor := Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	created, err := service.CreateApp(actor, "11111111-1111-1111-1111-111111111111", AppCreateRequest{
		Slug: "chatbot-web",
		Name: "Chatbot Web Widget",
		Type: "agent_consumer",
	})
	if err != nil {
		t.Fatalf("CreateApp() error = %v", err)
	}
	if created.APIKey == "" {
		t.Fatal("CreateApp() api key is empty")
	}

	apps, err := service.ListApps(actor, "11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("ListApps() error = %v", err)
	}
	if len(apps) < 2 {
		t.Fatalf("ListApps() len = %d, want at least 2", len(apps))
	}

	rotated, err := service.RotateAppKey(actor, created.TenantID, created.ID)
	if err != nil {
		t.Fatalf("RotateAppKey() error = %v", err)
	}
	if rotated.APIKey == "" || rotated.APIKey == created.APIKey {
		t.Fatal("RotateAppKey() did not issue a new key")
	}

	revoked, err := service.RevokeApp(actor, created.TenantID, created.ID)
	if err != nil {
		t.Fatalf("RevokeApp() error = %v", err)
	}
	if revoked.Status != "revoked" {
		t.Fatalf("RevokeApp() status = %q", revoked.Status)
	}
	if revoked.RevokedAt == nil {
		t.Fatal("RevokeApp() revoked_at is nil")
	}
}

func TestServiceCreateListAndRevokeGrant(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), SeedGrants())
	service := NewService(store, cache)
	actor := Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	if err := cache.SetJSON("acl:22222222-2222-2222-2222-222222222222:22222222-bbbb-2222-bbbb-222222222222", map[string]string{"cached": "yes"}, time.Minute); err != nil {
		t.Fatalf("SetJSON() error = %v", err)
	}

	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	created, err := service.CreateGrant(actor, GrantCreateRequest{
		GranteeTenantID: "22222222-2222-2222-2222-222222222222",
		GranteeAppID:    "22222222-bbbb-2222-bbbb-222222222222",
		ScopeType:       "domain",
		ScopeValue:      "noi_bo_hop_dong",
		Permission:      "write",
		ExpiresAt:       &expiresAt,
	})
	if err != nil {
		t.Fatalf("CreateGrant() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("grant id is empty")
	}

	var cached map[string]string
	if ok, _ := cache.GetJSON("acl:22222222-2222-2222-2222-222222222222:22222222-bbbb-2222-bbbb-222222222222", &cached); ok {
		t.Fatal("expected grant cache invalidation after create")
	}

	grants, err := service.ListGrants(actor, GrantListFilter{GrantorTenantID: actor.TenantID})
	if err != nil {
		t.Fatalf("ListGrants() error = %v", err)
	}
	if len(grants) < 1 {
		t.Fatalf("ListGrants() len = %d, want at least 1", len(grants))
	}

	revoked, err := service.RevokeGrant(actor, created.ID)
	if err != nil {
		t.Fatalf("RevokeGrant() error = %v", err)
	}
	if revoked.Status != "revoked" {
		t.Fatalf("RevokeGrant() status = %q", revoked.Status)
	}
	if revoked.RevokedAt == nil {
		t.Fatal("RevokeGrant() revoked_at is nil")
	}

	auditEntries, err := service.ListAuditLogs(actor, AuditFilter{
		ResourceOwnerTenantID: actor.TenantID,
	})
	if err != nil {
		t.Fatalf("ListAuditLogs() error = %v", err)
	}
	if len(auditEntries) < 2 {
		t.Fatalf("audit len = %d, want at least 2", len(auditEntries))
	}
	if auditEntries[0].Action != "grant.create" {
		t.Fatalf("first audit action = %q", auditEntries[0].Action)
	}
	if auditEntries[1].Action != "grant.revoke" {
		t.Fatalf("second audit action = %q", auditEntries[1].Action)
	}
}

func TestServiceRejectsCrossTenantWriteGrantWithoutExpiry(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), SeedGrants())
	service := NewService(store, cache)

	_, err := service.CreateGrant(Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, GrantCreateRequest{
		GranteeTenantID: "22222222-2222-2222-2222-222222222222",
		ScopeType:       "domain",
		ScopeValue:      "noi_bo_hop_dong",
		Permission:      "write",
	})
	if err == nil {
		t.Fatal("CreateGrant() error = nil, want bad request")
	}
}

func TestCreateGrantHandlerReturnsCreated(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), SeedGrants())
	handler := NewHandler(NewAccessResolver(store, store, cache), NewService(store, cache))

	req := httptest.NewRequest(http.MethodPost, "/v1/access/grants", strings.NewReader(`{"grantee_tenant_id":"22222222-2222-2222-2222-222222222222","grantee_app_id":"22222222-bbbb-2222-bbbb-222222222222","scope_type":"domain","scope_value":"noi_bo_hop_dong","permission":"read"}`))
	req = req.WithContext(ContextWithIdentity(req.Context(), Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}))
	rec := httptest.NewRecorder()

	handler.CreateGrant(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestListAuditRequiresOwnerScopedAdminAccess(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), SeedGrants())
	service := NewService(store, cache)
	adminActor := Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}
	service.RecordWriteAudit(adminActor, adminActor.TenantID, adminActor.AppID, "kg.node.create", "kg_node", "node_1", "allow", "", map[string]any{
		"domain_id": "noi_bo_hop_dong",
	})

	entries, err := service.ListAuditLogs(adminActor, AuditFilter{
		ResourceOwnerTenantID: adminActor.TenantID,
	})
	if err != nil {
		t.Fatalf("ListAuditLogs() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries len = %d, want 1", len(entries))
	}

	_, err = service.ListAuditLogs(Identity{
		TenantID: "22222222-2222-2222-2222-222222222222",
		AppID:    "22222222-bbbb-2222-bbbb-222222222222",
		AppType:  "agent_consumer",
	}, AuditFilter{
		ResourceOwnerTenantID: adminActor.TenantID,
	})
	if err == nil {
		t.Fatal("ListAuditLogs() error = nil, want forbidden")
	}
}

func TestListAuditHandlerReturnsListEnvelope(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), SeedGrants())
	service := NewService(store, cache)
	service.RecordWriteAudit(Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", "11111111-admin-1111-admin-111111111111", "kg.node.create", "kg_node", "node_1", "allow", "", nil)

	handler := NewHandler(NewAccessResolver(store, store, cache), service)
	req := httptest.NewRequest(http.MethodGet, "/v1/access/audit?resource_owner_tenant_id=11111111-1111-1111-1111-111111111111", nil)
	req = req.WithContext(ContextWithIdentity(req.Context(), Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}))
	rec := httptest.NewRecorder()

	handler.ListAudit(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"kg.node.create"`) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func mustNewCache(t *testing.T) *rediscache.Client {
	t.Helper()

	cache, err := rediscache.New(config.RedisConfig{
		Host: "127.0.0.1",
		Port: 6379,
		DB:   0,
	})
	if err != nil {
		t.Fatalf("rediscache.New() error = %v", err)
	}
	return &cache
}
