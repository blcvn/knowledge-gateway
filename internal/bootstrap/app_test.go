package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httptransport "github.com/go-kratos/kratos/v2/transport/http"
	"kg-service/internal/access"
	"kg-service/internal/config"
	"kg-service/internal/platform/postgres"
	"kg-service/internal/platform/rediscache"
)

func TestHealthzOmitsSensitiveConfiguration(t *testing.T) {
	app := &App{
		pg: postgres.Client{
			DSN: "postgres://kg:supersecret@db.internal:5432/kg_service?sslmode=require",
		},
		redis: rediscache.Client{
			Address: "cache.internal:6379",
			DB:      2,
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	app.handleHealthz(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if strings.Contains(rec.Body.String(), "supersecret") || strings.Contains(rec.Body.String(), "postgres://") {
		t.Fatalf("healthz body leaked sensitive config: %s", rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	postgresPayload, ok := payload["postgres"].(map[string]any)
	if !ok {
		t.Fatalf("postgres payload = %#v", payload["postgres"])
	}
	if _, exists := postgresPayload["dsn"]; exists {
		t.Fatalf("postgres payload = %#v, should not include dsn", postgresPayload)
	}
	if postgresPayload["max_open_conns"] == nil {
		t.Fatalf("postgres payload = %#v, want safe metadata", postgresPayload)
	}

	redisPayload, ok := payload["redis"].(map[string]any)
	if !ok {
		t.Fatalf("redis payload = %#v", payload["redis"])
	}
	if redisPayload["address"] != "cache.internal:6379" {
		t.Fatalf("redis address = %#v, want cache.internal:6379", redisPayload["address"])
	}
}

func TestHTTPToKratosHandlerCopiesMuxVarsToPathValue(t *testing.T) {
	server := httptransport.NewServer()
	router := server.Route("")

	const tenantID = "11111111-1111-1111-1111-111111111111"
	var gotTenantID string
	router.GET("/v1/tenants/{tenant_id}", httpToKratosHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTenantID = r.PathValue("tenant_id")
		w.WriteHeader(http.StatusNoContent)
	})))

	req := httptest.NewRequest(http.MethodGet, "/v1/tenants/"+tenantID, nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if gotTenantID != tenantID {
		t.Fatalf("tenant_id = %q, want %q", gotTenantID, tenantID)
	}
}

func TestTenantAppRoutesDispatchAndRouter404(t *testing.T) {
	app := newBootstrapAccessTestApp(t)
	server := httptransport.NewServer()
	app.registerRoutes(server.Route(""))

	wantRoutes := map[string]struct{}{
		http.MethodGet + " /v1/tenants/{tenant_id}":      {},
		http.MethodGet + " /v1/tenants/{tenant_id}/apps": {},
	}
	if err := server.WalkRoute(func(info httptransport.RouteInfo) error {
		delete(wantRoutes, info.Method+" "+info.Path)
		return nil
	}); err != nil {
		t.Fatalf("WalkRoute() error = %v", err)
	}
	if len(wantRoutes) > 0 {
		t.Fatalf("missing routes: %#v", wantRoutes)
	}

	const tenantID = "11111111-1111-1111-1111-111111111111"
	identityHeader := "Bearer kgsk_test_alpha_admin"

	tenantReq := httptest.NewRequest(http.MethodGet, "/v1/tenants/"+tenantID, nil)
	tenantReq.Header.Set("Authorization", identityHeader)
	tenantRec := httptest.NewRecorder()
	server.ServeHTTP(tenantRec, tenantReq)
	if tenantRec.Code != http.StatusOK {
		t.Fatalf("tenant status = %d body=%s", tenantRec.Code, tenantRec.Body.String())
	}
	var tenantPayload map[string]any
	if err := json.Unmarshal(tenantRec.Body.Bytes(), &tenantPayload); err != nil {
		t.Fatalf("json.Unmarshal() tenant error = %v", err)
	}
	if tenantPayload["id"] != tenantID {
		t.Fatalf("tenant id = %#v, want %q", tenantPayload["id"], tenantID)
	}

	appsReq := httptest.NewRequest(http.MethodGet, "/v1/tenants/"+tenantID+"/apps", nil)
	appsReq.Header.Set("Authorization", identityHeader)
	appsRec := httptest.NewRecorder()
	server.ServeHTTP(appsRec, appsReq)
	if appsRec.Code != http.StatusOK {
		t.Fatalf("apps status = %d body=%s", appsRec.Code, appsRec.Body.String())
	}
	if !strings.Contains(appsRec.Body.String(), `"data"`) {
		t.Fatalf("apps body = %s", appsRec.Body.String())
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/v1/tenants/"+tenantID+"/apps/extra", nil)
	missingReq.Header.Set("Authorization", identityHeader)
	missingRec := httptest.NewRecorder()
	server.ServeHTTP(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d body=%s", missingRec.Code, missingRec.Body.String())
	}
}

func newBootstrapAccessTestApp(t *testing.T) *App {
	t.Helper()

	cache := mustNewBootstrapCache(t)
	store := access.NewMemoryStore()
	store.Seed(access.SeedTenants(), access.SeedApps(), access.SeedGrants())

	return &App{
		accessHandler:    access.NewHandler(access.NewAccessResolver(store, store, cache), access.NewService(store, cache)),
		accessMiddleware: access.NewMiddleware(access.NewIdentityResolver(store, cache)),
	}
}

func mustNewBootstrapCache(t *testing.T) *rediscache.Client {
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
