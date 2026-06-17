package access

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"kg-service/internal/config"
	"kg-service/internal/httpapi/respond"
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

func TestMiddlewareRateLimitsByTenantTier(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), nil)

	limiter := NewRateLimiter(store, map[string]int{"pro": 1})
	limiter.SetNow(func() time.Time { return time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC) })
	middleware := NewMiddleware(NewIdentityResolver(store, cache), limiter)

	handler := middleware.RequireIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for i := 0; i < 2; i++ {
		request := httptest.NewRequest(http.MethodPost, "/v1/kg/write/nodes", strings.NewReader(`{"foo":"bar"}`))
		request.Header.Set("Authorization", "Bearer kgsk_test_alpha")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if i == 0 && recorder.Code != http.StatusNoContent {
			t.Fatalf("first request status = %d", recorder.Code)
		}
		if i == 1 {
			if recorder.Code != http.StatusTooManyRequests {
				t.Fatalf("second request status = %d, want %d", recorder.Code, http.StatusTooManyRequests)
			}
			if !strings.Contains(recorder.Body.String(), respond.CodeTooManyRequests) {
				t.Fatalf("second response = %s", recorder.Body.String())
			}
		}
	}
}

func TestIdentityResolverRejectsRevokedAppAndExpiredGrants(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), SeedGrants())

	identityResolver := NewIdentityResolver(store, cache)
	service := NewService(store, cache)

	identity, err := identityResolver.Resolve("Bearer kgsk_test_alpha")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if identity.AppID == "" {
		t.Fatal("Resolve() app id is empty")
	}

	if _, err := service.RevokeApp(Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", "11111111-aaaa-1111-aaaa-111111111111"); err != nil {
		t.Fatalf("RevokeApp() error = %v", err)
	}

	if _, err := identityResolver.Resolve("Bearer kgsk_test_alpha"); err == nil {
		t.Fatal("Resolve() error = nil, want revoked key rejection")
	}

	cache2 := mustNewCache(t)
	accessResolver := NewAccessResolver(store, store, cache2)
	store.CreateTenant(Tenant{
		ID:                   "33333333-3333-3333-3333-333333333333",
		Slug:                 "test-gamma",
		Name:                 "Test Gamma Tenant",
		Status:               "active",
		Tier:                 "pro",
		DefaultSharingPolicy: "deny_all",
	})
	store.CreateApp(App{
		ID:           "33333333-aaaa-3333-aaaa-333333333333",
		TenantID:     "33333333-3333-3333-3333-333333333333",
		Slug:         "test-gamma-app",
		Name:         "Test Gamma App",
		Type:         "agent_consumer",
		APIKeyHash:   APIKeyHash("kgsk_test_gamma"),
		APIKeyPrefix: "kgsk_tes",
		Status:       "active",
		CreatedAt:    time.Now().UTC(),
	})
	expiredGrant := time.Now().UTC().Add(-time.Hour)
	store.CreateGrant(AccessGrant{
		ID:              "expired-grant",
		GrantorTenantID: "22222222-2222-2222-2222-222222222222",
		GrantorAppID:    "22222222-bbbb-2222-bbbb-222222222222",
		GranteeTenantID: "33333333-3333-3333-3333-333333333333",
		GranteeAppID:    "33333333-aaaa-3333-aaaa-333333333333",
		ScopeType:       "domain",
		ScopeValue:      "shared-domain",
		Permission:      "read",
		Status:          "active",
		ExpiresAt:       &expiredGrant,
		CreatedAt:       time.Now().UTC().Add(-2 * time.Hour),
	})
	owners, err := accessResolver.ResolveVisibleOwners(Identity{
		TenantID: "33333333-3333-3333-3333-333333333333",
		AppID:    "33333333-aaaa-3333-aaaa-333333333333",
	})
	if err != nil {
		t.Fatalf("ResolveVisibleOwners() error = %v", err)
	}
	for _, owner := range owners {
		if owner.Source == "grant" && owner.ScopeValue == "shared-domain" && owner.TenantID == "22222222-2222-2222-2222-222222222222" {
			t.Fatalf("expired grant unexpectedly visible in owners: %#v", owners)
		}
	}
}

func TestMiddlewareUsesFreshIdentityPerRequest(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), nil)

	middleware := NewMiddleware(NewIdentityResolver(store, cache))
	var seen []Identity
	handler := middleware.RequireIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := IdentityFromContext(r.Context())
		if !ok {
			t.Fatal("identity missing from context")
		}
		seen = append(seen, identity)
		w.WriteHeader(http.StatusNoContent)
	}))

	for _, token := range []string{"Bearer kgsk_test_alpha", "Bearer kgsk_test_beta"} {
		req := httptest.NewRequest(http.MethodGet, "/v1/access/resolve", nil)
		req.Header.Set("Authorization", token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
		}
	}

	if len(seen) != 2 {
		t.Fatalf("seen len = %d, want 2", len(seen))
	}
	if seen[0].AppID == seen[1].AppID {
		t.Fatalf("expected fresh identity per request, got %#v", seen)
	}
}

func TestAccessResolverConcurrentRequestsStayTenantScoped(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), SeedGrants())
	store.CreateGrant(AccessGrant{
		ID:              "grant-alpha-to-beta",
		GrantorTenantID: "11111111-1111-1111-1111-111111111111",
		GrantorAppID:    "11111111-admin-1111-admin-111111111111",
		GranteeTenantID: "22222222-2222-2222-2222-222222222222",
		GranteeAppID:    "22222222-bbbb-2222-bbbb-222222222222",
		ScopeType:       "domain",
		ScopeValue:      "shared-domain",
		Permission:      "read",
		Status:          "active",
	})

	resolver := NewAccessResolver(store, store, cache)
	alpha := Identity{TenantID: "11111111-1111-1111-1111-111111111111", AppID: "11111111-aaaa-1111-aaaa-111111111111"}
	beta := Identity{TenantID: "22222222-2222-2222-2222-222222222222", AppID: "22222222-bbbb-2222-bbbb-222222222222"}

	type result struct {
		identity Identity
		owners   []VisibleOwner
		err      error
	}
	results := make(chan result, 20)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		for _, identity := range []Identity{alpha, beta} {
			wg.Add(1)
			go func(identity Identity) {
				defer wg.Done()
				owners, err := resolver.ResolveVisibleOwners(identity)
				results <- result{identity: identity, owners: owners, err: err}
			}(identity)
		}
	}
	wg.Wait()
	close(results)

	for res := range results {
		if res.err != nil {
			t.Fatalf("ResolveVisibleOwners(%#v) error = %v", res.identity, res.err)
		}
		switch res.identity.TenantID {
		case alpha.TenantID:
			if !containsGrantFrom(res.owners, "22222222-2222-2222-2222-222222222222") {
				t.Fatalf("alpha owners = %#v", res.owners)
			}
			if containsGrantFrom(res.owners, "99999999-9999-9999-9999-999999999999") {
				t.Fatalf("alpha owners leaked foreign grant: %#v", res.owners)
			}
		case beta.TenantID:
			if !containsGrantFrom(res.owners, "11111111-1111-1111-1111-111111111111") {
				t.Fatalf("beta owners = %#v", res.owners)
			}
		}
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

func TestListAppsReturnsStandardListEnvelope(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), nil)

	handler := NewHandler(NewAccessResolver(store, store, cache), NewService(store, cache))
	request := httptest.NewRequest(http.MethodGet, "/v1/tenants/11111111-1111-1111-1111-111111111111/apps?limit=1", nil)
	request.SetPathValue("tenant_id", "11111111-1111-1111-1111-111111111111")
	request = request.WithContext(ContextWithIdentity(request.Context(), Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}))
	recorder := httptest.NewRecorder()

	handler.ListApps(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload respond.ListEnvelope[AppResponse]
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(payload.Data))
	}
	if payload.NextCursor == "" {
		t.Fatalf("next_cursor = %q, want non-empty", payload.NextCursor)
	}
	if !payload.HasMore {
		t.Fatal("has_more = false, want true")
	}

	nextRequest := httptest.NewRequest(http.MethodGet, "/v1/tenants/11111111-1111-1111-1111-111111111111/apps?limit=1&cursor="+payload.NextCursor, nil)
	nextRequest.SetPathValue("tenant_id", "11111111-1111-1111-1111-111111111111")
	nextRequest = nextRequest.WithContext(ContextWithIdentity(nextRequest.Context(), Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}))
	nextRecorder := httptest.NewRecorder()

	handler.ListApps(nextRecorder, nextRequest)

	if nextRecorder.Code != http.StatusOK {
		t.Fatalf("next status = %d body=%s", nextRecorder.Code, nextRecorder.Body.String())
	}
	var nextPayload respond.ListEnvelope[AppResponse]
	if err := json.Unmarshal(nextRecorder.Body.Bytes(), &nextPayload); err != nil {
		t.Fatalf("json.Unmarshal() next error = %v", err)
	}
	if len(nextPayload.Data) != 1 {
		t.Fatalf("next data len = %d, want 1", len(nextPayload.Data))
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

	auditEntries, err := service.ListAuditLogs(actor, AuditListFilter{
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

func TestGrantLifecyclePropagatesToVisibilityAndAuditTrail(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), SeedGrants())
	service := NewService(store, cache)
	resolver := NewAccessResolver(store, store, cache)
	actor := Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	created, err := service.CreateGrant(actor, GrantCreateRequest{
		GranteeTenantID: "22222222-2222-2222-2222-222222222222",
		GranteeAppID:    "22222222-bbbb-2222-bbbb-222222222222",
		ScopeType:       "domain",
		ScopeValue:      "shared-domain",
		Permission:      "read",
		ExpiresAt:       &expiresAt,
	})
	if err != nil {
		t.Fatalf("CreateGrant() error = %v", err)
	}

	owners, err := resolver.ResolveVisibleOwners(Identity{
		TenantID: "22222222-2222-2222-2222-222222222222",
		AppID:    "22222222-bbbb-2222-bbbb-222222222222",
	})
	if err != nil {
		t.Fatalf("ResolveVisibleOwners() after create error = %v", err)
	}
	if !hasGrantOwner(owners, created.GrantorTenantID, created.GrantorAppID, created.ScopeType, created.ScopeValue, created.Permission) {
		t.Fatalf("owners after create = %#v", owners)
	}

	revoked, err := service.RevokeGrant(actor, created.ID)
	if err != nil {
		t.Fatalf("RevokeGrant() error = %v", err)
	}
	if revoked.Status != "revoked" || revoked.RevokedAt == nil {
		t.Fatalf("revoked grant = %#v", revoked)
	}

	owners, err = resolver.ResolveVisibleOwners(Identity{
		TenantID: "22222222-2222-2222-2222-222222222222",
		AppID:    "22222222-bbbb-2222-bbbb-222222222222",
	})
	if err != nil {
		t.Fatalf("ResolveVisibleOwners() after revoke error = %v", err)
	}
	if hasGrantOwner(owners, created.GrantorTenantID, created.GrantorAppID, created.ScopeType, created.ScopeValue, created.Permission) {
		t.Fatalf("owners after revoke = %#v", owners)
	}

	auditEntries, err := service.ListAuditLogs(actor, AuditListFilter{ResourceOwnerTenantID: actor.TenantID})
	if err != nil {
		t.Fatalf("ListAuditLogs() error = %v", err)
	}
	if len(auditEntries) < 2 {
		t.Fatalf("audit len = %d, want at least 2", len(auditEntries))
	}
	if auditEntries[0].Action != "grant.create" || auditEntries[1].Action != "grant.revoke" {
		t.Fatalf("audit actions = %q, %q", auditEntries[0].Action, auditEntries[1].Action)
	}
}

func TestGrantRevokeInvalidatesWarmAclCacheImmediately(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), SeedGrants())
	service := NewService(store, cache)
	resolver := NewAccessResolver(store, store, cache)
	actor := Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	created, err := service.CreateGrant(actor, GrantCreateRequest{
		GranteeTenantID: "22222222-2222-2222-2222-222222222222",
		GranteeAppID:    "22222222-bbbb-2222-bbbb-222222222222",
		ScopeType:       "domain",
		ScopeValue:      "shared-domain",
		Permission:      "read",
		ExpiresAt:       &expiresAt,
	})
	if err != nil {
		t.Fatalf("CreateGrant() error = %v", err)
	}

	identity := Identity{
		TenantID: "22222222-2222-2222-2222-222222222222",
		AppID:    "22222222-bbbb-2222-bbbb-222222222222",
	}
	owners, err := resolver.ResolveVisibleOwners(identity)
	if err != nil {
		t.Fatalf("ResolveVisibleOwners() warm cache error = %v", err)
	}
	if !hasGrantOwner(owners, created.GrantorTenantID, created.GrantorAppID, created.ScopeType, created.ScopeValue, created.Permission) {
		t.Fatalf("owners before revoke = %#v", owners)
	}

	if _, err := service.RevokeGrant(actor, created.ID); err != nil {
		t.Fatalf("RevokeGrant() error = %v", err)
	}

	owners, err = resolver.ResolveVisibleOwners(identity)
	if err != nil {
		t.Fatalf("ResolveVisibleOwners() post revoke error = %v", err)
	}
	if hasGrantOwner(owners, created.GrantorTenantID, created.GrantorAppID, created.ScopeType, created.ScopeValue, created.Permission) {
		t.Fatalf("owners after revoke = %#v", owners)
	}
}

func TestListGrantsSupportsFilterAndPagination(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), SeedGrants())
	store.CreateGrant(AccessGrant{
		ID:              "grant-alpha-read-beta-domain-2",
		GrantorTenantID: "22222222-2222-2222-2222-222222222222",
		GrantorAppID:    "22222222-bbbb-2222-bbbb-222222222222",
		GranteeTenantID: "11111111-1111-1111-1111-111111111111",
		GranteeAppID:    "11111111-admin-1111-admin-111111111111",
		ScopeType:       "domain",
		ScopeValue:      "shared-domain",
		Permission:      "search",
		Status:          "active",
		CreatedAt:       time.Date(2026, 6, 17, 11, 0, 0, 0, time.UTC),
	})

	handler := NewHandler(NewAccessResolver(store, store, cache), NewService(store, cache))
	req := httptest.NewRequest(http.MethodGet, "/v1/access/grants?grantor_tenant_id=22222222-2222-2222-2222-222222222222&limit=1", nil)
	req = req.WithContext(ContextWithIdentity(req.Context(), Identity{
		TenantID: "22222222-2222-2222-2222-222222222222",
		AppID:    "22222222-bbbb-2222-bbbb-222222222222",
		AppType:  "admin_tool",
	}))
	rec := httptest.NewRecorder()

	handler.ListGrants(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var payload respond.ListEnvelope[GrantResponse]
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("data len = %d, want 1", len(payload.Data))
	}
	if payload.NextCursor == "" || !payload.HasMore {
		t.Fatalf("payload = %#v", payload)
	}

	nextReq := httptest.NewRequest(http.MethodGet, "/v1/access/grants?grantor_tenant_id=22222222-2222-2222-2222-222222222222&limit=1&cursor="+payload.NextCursor, nil)
	nextReq = nextReq.WithContext(ContextWithIdentity(nextReq.Context(), Identity{
		TenantID: "22222222-2222-2222-2222-222222222222",
		AppID:    "22222222-bbbb-2222-bbbb-222222222222",
		AppType:  "admin_tool",
	}))
	nextRec := httptest.NewRecorder()

	handler.ListGrants(nextRec, nextReq)

	if nextRec.Code != http.StatusOK {
		t.Fatalf("next status = %d body=%s", nextRec.Code, nextRec.Body.String())
	}
	var nextPayload respond.ListEnvelope[GrantResponse]
	if err := json.Unmarshal(nextRec.Body.Bytes(), &nextPayload); err != nil {
		t.Fatalf("json.Unmarshal() next error = %v", err)
	}
	if len(nextPayload.Data) != 1 {
		t.Fatalf("next data len = %d, want 1", len(nextPayload.Data))
	}
}

func TestListHandlersRejectMalformedTenantFilters(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), SeedGrants())
	handler := NewHandler(NewAccessResolver(store, store, cache), NewService(store, cache))
	identity := Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	cases := []struct {
		name string
		path string
	}{
		{
			name: "grants",
			path: "/v1/access/grants?grantor_tenant_id=not-a-uuid",
		},
		{
			name: "audit",
			path: "/v1/access/audit?resource_owner_tenant_id=not-a-uuid",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req = req.WithContext(ContextWithIdentity(req.Context(), identity))
			rec := httptest.NewRecorder()

			switch tc.name {
			case "grants":
				handler.ListGrants(rec, req)
			case "audit":
				handler.ListAudit(rec, req)
			}

			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), respond.CodeValidationFailed) {
				t.Fatalf("body = %s", rec.Body.String())
			}
		})
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

func TestServiceRejectsForeignGrantorAppPrivilegeEscalation(t *testing.T) {
	cache := mustNewCache(t)
	store := NewMemoryStore()
	store.Seed(SeedTenants(), SeedApps(), SeedGrants())
	service := NewService(store, cache)

	_, err := service.CreateGrant(Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, GrantCreateRequest{
		GrantorAppID:    "22222222-bbbb-2222-bbbb-222222222222",
		GranteeTenantID: "22222222-2222-2222-2222-222222222222",
		ScopeType:       "domain",
		ScopeValue:      "shared-domain",
		Permission:      "read",
	})
	if err == nil {
		t.Fatal("CreateGrant() error = nil, want not found for foreign grantor app")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("CreateGrant() error = %v, want not found", err)
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

	entries, err := service.ListAuditLogs(adminActor, AuditListFilter{
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
	}, AuditListFilter{
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

func hasGrantOwner(owners []VisibleOwner, tenantID, appID, scopeType, scopeValue, permission string) bool {
	for _, owner := range owners {
		if owner.TenantID != tenantID || owner.ScopeType != scopeType || owner.ScopeValue != scopeValue || owner.Permission != permission || owner.Source != "grant" {
			continue
		}
		if owner.AppID == appID || (owner.AppID == "*" && strings.TrimSpace(appID) == "") {
			return true
		}
	}
	return false
}

func containsGrantFrom(owners []VisibleOwner, tenantID string) bool {
	for _, owner := range owners {
		if owner.Source == "grant" && owner.TenantID == tenantID {
			return true
		}
	}
	return false
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
