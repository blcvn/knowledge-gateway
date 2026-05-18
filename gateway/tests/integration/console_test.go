package integration_test

import (
	"net/http"
	"testing"

	"github.com/vnp-community/vnp-memory/gateway/domain"
)

// Console Integration Tests (T15/SOL-002)
// Note: Console endpoints enforce admin role via requireAdmin().
// Since integration tests don't wire auth middleware, console handlers
// correctly return 401 UNAUTHENTICATED for raw HTTP requests.
// These tests validate:
// 1. Route registration (route exists, not 404)
// 2. Auth enforcement (returns 401, not 404)
// 3. Service proxying when auth is bypassed (via direct handler unit tests)

// ── Route Registration Validation ──────────────────────────────
// All console routes should return 401 (auth required), NOT 404 (route not found).

func TestConsole_Routes_Dashboard(t *testing.T) {
	srv, reg := setupTestServer(t)
	reg.services["vnp-platform"] = &domain.RouteTarget{Service: "vnp-platform", Address: "localhost:9043"}

	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/v1/console/dashboard/health"},
		{"GET", "/v1/console/dashboard/metrics"},
		{"GET", "/v1/console/dashboard/throughput"},
		{"GET", "/v1/console/dashboard/heatmap"},
	}
	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			resp, body := doGet(t, srv, r.path)
			// Must be 401 (route exists, auth required) — NOT 404
			assertStatus(t, resp, http.StatusUnauthorized)
			assertContains(t, body, "UNAUTHENTICATED")
		})
	}
}

func TestConsole_Routes_Explorer(t *testing.T) {
	srv, _ := setupTestServer(t)
	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/v1/console/memory/mem-123"},
		{"GET", "/v1/console/memory/mem-123/neighbors"},
		{"GET", "/v1/console/memory/mem-123/versions"},
	}
	for _, r := range routes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			resp, body := doGet(t, srv, r.path)
			assertStatus(t, resp, http.StatusUnauthorized)
			assertContains(t, body, "UNAUTHENTICATED")
		})
	}

	resp, body := doPost(t, srv, "/v1/console/memory/search", `{"query":"test"}`)
	assertStatus(t, resp, http.StatusUnauthorized)
	assertContains(t, body, "UNAUTHENTICATED")
}

func TestConsole_Routes_Graph(t *testing.T) {
	srv, _ := setupTestServer(t)
	getRoutes := []string{
		"/v1/console/graph/entity/e1",
		"/v1/console/graph/ontology",
	}
	for _, path := range getRoutes {
		t.Run("GET "+path, func(t *testing.T) {
			resp, body := doGet(t, srv, path)
			assertStatus(t, resp, http.StatusUnauthorized)
			assertContains(t, body, "UNAUTHENTICATED")
		})
	}

	postRoutes := []string{
		"/v1/console/graph/subgraph",
		"/v1/console/graph/timeline",
		"/v1/console/graph/query",
	}
	for _, path := range postRoutes {
		t.Run("POST "+path, func(t *testing.T) {
			resp, body := doPost(t, srv, path, `{}`)
			assertStatus(t, resp, http.StatusUnauthorized)
			assertContains(t, body, "UNAUTHENTICATED")
		})
	}
}

func TestConsole_Routes_Profile(t *testing.T) {
	srv, _ := setupTestServer(t)
	routes := []string{
		"/v1/console/profiles",
		"/v1/console/profiles/config",
		"/v1/console/profiles/user-1",
		"/v1/console/profiles/user-1/events",
		"/v1/console/profiles/user-1/context",
		"/v1/console/profiles/user-1/buffers",
	}
	for _, path := range routes {
		t.Run("GET "+path, func(t *testing.T) {
			resp, body := doGet(t, srv, path)
			assertStatus(t, resp, http.StatusUnauthorized)
			assertContains(t, body, "UNAUTHENTICATED")
		})
	}
}

func TestConsole_Routes_Adaptive(t *testing.T) {
	srv, _ := setupTestServer(t)
	routes := []string{
		"/v1/console/adaptive/memories",
		"/v1/console/adaptive/connectors",
		"/v1/console/adaptive/analytics",
		"/v1/console/adaptive/forget-rules",
	}
	for _, path := range routes {
		t.Run("GET "+path, func(t *testing.T) {
			resp, body := doGet(t, srv, path)
			assertStatus(t, resp, http.StatusUnauthorized)
			assertContains(t, body, "UNAUTHENTICATED")
		})
	}
}

func TestConsole_Routes_Debugger(t *testing.T) {
	srv, _ := setupTestServer(t)

	resp, body := doGet(t, srv, "/v1/console/debugger/traces")
	assertStatus(t, resp, http.StatusUnauthorized)
	assertContains(t, body, "UNAUTHENTICATED")

	resp, body = doPost(t, srv, "/v1/console/debugger/trace", `{"query":"test"}`)
	assertStatus(t, resp, http.StatusUnauthorized)
	assertContains(t, body, "UNAUTHENTICATED")
}

func TestConsole_Routes_Sessions(t *testing.T) {
	srv, reg := setupTestServer(t)
	reg.services["zep-core"] = &domain.RouteTarget{Service: "zep-core", Address: "localhost:9067"}

	routes := []string{
		"/v1/console/sessions",
		"/v1/console/sessions/live",
		"/v1/console/sessions/sess-1",
		"/v1/console/sessions/sess-1/timeline",
		"/v1/console/sessions/sess-1/diff",
		"/v1/console/sessions/sess-1/working-memory",
		"/v1/console/sessions/sess-1/user-summary",
	}
	for _, path := range routes {
		t.Run("GET "+path, func(t *testing.T) {
			resp, body := doGet(t, srv, path)
			assertStatus(t, resp, http.StatusUnauthorized)
			assertContains(t, body, "UNAUTHENTICATED")
		})
	}
}

func TestConsole_Routes_Governance(t *testing.T) {
	srv, _ := setupTestServer(t)
	routes := []string{
		"/v1/console/governance/tenants",
		"/v1/console/governance/policies",
		"/v1/console/governance/audit",
	}
	for _, path := range routes {
		t.Run("GET "+path, func(t *testing.T) {
			resp, body := doGet(t, srv, path)
			assertStatus(t, resp, http.StatusUnauthorized)
			assertContains(t, body, "UNAUTHENTICATED")
		})
	}
}

func TestConsole_Routes_Pipelines(t *testing.T) {
	srv, reg := setupTestServer(t)
	reg.services["vnp-platform"] = &domain.RouteTarget{Service: "vnp-platform", Address: "localhost:9043"}

	routes := []string{
		"/v1/console/pipelines/status",
		"/v1/console/pipelines/queues",
		"/v1/console/pipelines/workers",
		"/v1/console/pipelines/templates",
		"/v1/console/pipelines/cognee",
	}
	for _, path := range routes {
		t.Run("GET "+path, func(t *testing.T) {
			resp, body := doGet(t, srv, path)
			assertStatus(t, resp, http.StatusUnauthorized)
			assertContains(t, body, "UNAUTHENTICATED")
		})
	}
}

func TestConsole_Routes_Infra(t *testing.T) {
	srv, reg := setupTestServer(t)
	reg.services["vnp-platform"] = &domain.RouteTarget{Service: "vnp-platform", Address: "localhost:9043"}

	routes := []string{
		"/v1/console/infra/topology",
		"/v1/console/infra/services",
		"/v1/console/infra/databases",
		"/v1/console/infra/resources",
		"/v1/console/infra/deployments",
	}
	for _, path := range routes {
		t.Run("GET "+path, func(t *testing.T) {
			resp, body := doGet(t, srv, path)
			assertStatus(t, resp, http.StatusUnauthorized)
			assertContains(t, body, "UNAUTHENTICATED")
		})
	}
}

func TestConsole_Routes_Observability(t *testing.T) {
	srv, reg := setupTestServer(t)
	reg.services["vnp-platform"] = &domain.RouteTarget{Service: "vnp-platform", Address: "localhost:9043"}

	routes := []string{
		"/v1/console/observability/metrics",
		"/v1/console/observability/traces",
		"/v1/console/observability/errors",
		"/v1/console/observability/costs",
	}
	for _, path := range routes {
		t.Run("GET "+path, func(t *testing.T) {
			resp, body := doGet(t, srv, path)
			assertStatus(t, resp, http.StatusUnauthorized)
			assertContains(t, body, "UNAUTHENTICATED")
		})
	}
}

func TestConsole_Routes_WS(t *testing.T) {
	srv, _ := setupTestServer(t)
	resp, body := doGet(t, srv, "/v1/console/ws")
	assertStatus(t, resp, http.StatusUnauthorized)
	assertContains(t, body, "UNAUTHENTICATED")
}

func TestConsole_Routes_GDPR(t *testing.T) {
	srv, _ := setupTestServer(t)
	routes := []struct {
		path string
		body string
	}{
		{"/v1/console/governance/gdpr/forget", `{"user_id":"u-123"}`},
		{"/v1/console/governance/gdpr/forget/preview", `{"user_id":"u-123"}`},
	}
	for _, r := range routes {
		t.Run("POST "+r.path, func(t *testing.T) {
			resp, body := doPost(t, srv, r.path, r.body)
			// Governance endpoints require super_admin
			assertStatus(t, resp, http.StatusUnauthorized)
			assertContains(t, body, "UNAUTHENTICATED")
		})
	}
}

// ── Console Route NOT 404 ──────────────────────────────────────
// Verify that console routes return auth errors, not 404.

func TestConsole_Routes_NotFound(t *testing.T) {
	srv, _ := setupTestServer(t)
	// These should return 404 (unknown routes)
	routes := []string{
		"/v1/console/nonexistent",
		"/v1/console/dashboard/nonexistent",
	}
	for _, path := range routes {
		t.Run("GET "+path, func(t *testing.T) {
			resp, body := doGet(t, srv, path)
			assertStatus(t, resp, http.StatusNotFound)
			assertContains(t, body, "NOT_FOUND")
		})
	}
}
