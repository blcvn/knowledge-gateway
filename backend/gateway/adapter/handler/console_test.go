package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vnp-community/vnp-memory/gateway/domain"
	"github.com/vnp-community/vnp-memory/gateway/infra/middleware"
)

// mockAdminContext injects an admin auth context into the request.
func mockAdminContext(r *http.Request) *http.Request {
	ctx := middleware.WithAuthContext(r.Context(), &domain.AuthContext{
		TenantID: "test-tenant",
		UserID:   "admin-user",
		Roles:    []string{"admin"},
		Scopes:   []string{"*"},
		RateTier: domain.RateTierEnterprise,
	})
	return r.WithContext(ctx)
}

// mockSuperAdminContext injects a super_admin auth context.
func mockSuperAdminContext(r *http.Request) *http.Request {
	ctx := middleware.WithAuthContext(r.Context(), &domain.AuthContext{
		TenantID: "test-tenant",
		UserID:   "super-admin",
		Roles:    []string{"super_admin"},
		Scopes:   []string{"*"},
		RateTier: domain.RateTierEnterprise,
	})
	return r.WithContext(ctx)
}

// mockUserContext injects a non-admin auth context.
func mockUserContext(r *http.Request) *http.Request {
	ctx := middleware.WithAuthContext(r.Context(), &domain.AuthContext{
		TenantID: "test-tenant",
		UserID:   "regular-user",
		Roles:    []string{"user"},
		Scopes:   []string{"read"},
		RateTier: domain.RateTierFree,
	})
	return r.WithContext(ctx)
}

func TestRequireAdmin_WithAdminRole(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v1/console/dashboard/health", nil)
	r = mockAdminContext(r)
	if !requireAdmin(w, r) {
		t.Error("expected requireAdmin to return true for admin role")
	}
}

func TestRequireAdmin_WithSuperAdminRole(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v1/console/dashboard/health", nil)
	r = mockSuperAdminContext(r)
	if !requireAdmin(w, r) {
		t.Error("expected requireAdmin to return true for super_admin role")
	}
}

func TestRequireAdmin_WithUserRole_Rejected(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v1/console/dashboard/health", nil)
	r = mockUserContext(r)
	if requireAdmin(w, r) {
		t.Error("expected requireAdmin to return false for user role")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestRequireAdmin_NoAuth_Rejected(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v1/console/dashboard/health", nil)
	if requireAdmin(w, r) {
		t.Error("expected requireAdmin to return false without auth context")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestRequireSuperAdmin_WithSuperAdmin(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/v1/console/governance/tenants", nil)
	r = mockSuperAdminContext(r)
	if !requireSuperAdmin(w, r) {
		t.Error("expected requireSuperAdmin to return true for super_admin")
	}
}

func TestRequireSuperAdmin_WithAdmin_Rejected(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/v1/console/governance/tenants", nil)
	r = mockAdminContext(r)
	if requireSuperAdmin(w, r) {
		t.Error("expected requireSuperAdmin to return false for admin (not super_admin)")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestDashboardHandler_Health_AdminOK(t *testing.T) {
	reg := &noopTestRegistry{}
	h := NewDashboardHandler(reg, testLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v1/console/dashboard/health", nil)
	r = mockAdminContext(r)
	h.Health(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestDashboardHandler_Health_NoAuth(t *testing.T) {
	reg := &noopTestRegistry{}
	h := NewDashboardHandler(reg, testLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v1/console/dashboard/health", nil)
	h.Health(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestExplorerHandler_Search_AdminOK(t *testing.T) {
	reg := &noopTestRegistry{}
	h := NewExplorerHandler(reg, testLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/v1/console/memory/search", nil)
	r = mockAdminContext(r)
	h.Search(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGovernanceHandler_ListTenants_RequiresSuperAdmin(t *testing.T) {
	reg := &noopTestRegistry{}
	h := NewGovernanceHandler(reg, testLogger())

	// Admin (not super_admin) should be rejected
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v1/console/governance/tenants", nil)
	r = mockAdminContext(r)
	h.ListTenants(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for admin (not super_admin), got %d", w.Code)
	}

	// Super admin should pass
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/v1/console/governance/tenants", nil)
	r = mockSuperAdminContext(r)
	h.ListTenants(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for super_admin, got %d", w.Code)
	}
}

func TestGraphHandler_Subgraph_AdminOK(t *testing.T) {
	reg := &noopTestRegistry{}
	h := NewGraphHandler(reg, testLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/v1/console/graph/subgraph", nil)
	r = mockAdminContext(r)
	h.Subgraph(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestSessionHandler_ListSessions_AdminOK(t *testing.T) {
	reg := &noopTestRegistry{}
	h := NewSessionHandler(reg, testLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v1/console/sessions", nil)
	r = mockAdminContext(r)
	h.ListSessions(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestPipelineHandler_Status_AdminOK(t *testing.T) {
	reg := &noopTestRegistry{}
	h := NewPipelineHandler(reg, testLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v1/console/pipelines/status", nil)
	r = mockAdminContext(r)
	h.Status(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestInfraHandler_Topology_AdminOK(t *testing.T) {
	reg := &noopTestRegistry{}
	h := NewInfraHandler(reg, testLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v1/console/infra/topology", nil)
	r = mockAdminContext(r)
	h.Topology(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestObservabilityHandler_Metrics_AdminOK(t *testing.T) {
	reg := &noopTestRegistry{}
	h := NewObservabilityHandler(reg, testLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v1/console/observability/metrics", nil)
	r = mockAdminContext(r)
	h.Metrics(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestWSHandler_HandleWS_NoAuth(t *testing.T) {
	h := NewWSHandler(testLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v1/console/ws", nil)
	h.HandleWS(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", w.Code)
	}
}

func TestWSHandler_HandleWS_NonAdmin(t *testing.T) {
	h := NewWSHandler(testLogger())

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v1/console/ws", nil)
	r = mockUserContext(r)
	h.HandleWS(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for non-admin, got %d", w.Code)
	}
}

func TestWSHandler_ConnectionCount(t *testing.T) {
	h := NewWSHandler(testLogger())
	if h.ConnectionCount() != 0 {
		t.Errorf("expected 0 connections initially, got %d", h.ConnectionCount())
	}
}

// TestAllConsoleHandlers_RoleEnforcement verifies all console handler types reject unauthenticated.
func TestAllConsoleHandlers_RoleEnforcement(t *testing.T) {
	reg := &noopTestRegistry{}
	log := testLogger()

	handlers := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"Dashboard.Health", NewDashboardHandler(reg, log).Health},
		{"Dashboard.Metrics", NewDashboardHandler(reg, log).Metrics},
		{"Dashboard.Throughput", NewDashboardHandler(reg, log).Throughput},
		{"Dashboard.Heatmap", NewDashboardHandler(reg, log).Heatmap},
		{"Explorer.Search", NewExplorerHandler(reg, log).Search},
		{"Explorer.GetMemory", NewExplorerHandler(reg, log).GetMemory},
		{"Graph.Subgraph", NewGraphHandler(reg, log).Subgraph},
		{"Graph.GetEntity", NewGraphHandler(reg, log).GetEntity},
		{"Profile.ListProfiles", NewProfileHandler(reg, log).ListProfiles},
		{"Adaptive.ListMemories", NewAdaptiveHandler(reg, log).ListMemories},
		{"Debugger.CreateTrace", NewDebuggerHandler(reg, log).CreateTrace},
		{"Session.ListSessions", NewSessionHandler(reg, log).ListSessions},
		{"Pipeline.Status", NewPipelineHandler(reg, log).Status},
		{"Infra.Topology", NewInfraHandler(reg, log).Topology},
		{"Observability.Metrics", NewObservabilityHandler(reg, log).Metrics},
	}

	for _, tc := range handlers {
		t.Run(tc.name+"_NoAuth", func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/", nil)
			tc.handler(w, r)

			var resp map[string]any
			json.NewDecoder(w.Body).Decode(&resp)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("%s: expected 401, got %d", tc.name, w.Code)
			}
		})
	}
}
