package integration_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/vnp-community/vnp-memory/gateway/adapter/handler"
	"github.com/vnp-community/vnp-memory/gateway/domain"
	"github.com/vnp-community/vnp-memory/gateway/usecase"
)

// mockRegistry implements port.ServiceRegistry for testing.
type mockRegistry struct {
	services map[string]*domain.RouteTarget
	response []byte
	err      error
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{
		services: map[string]*domain.RouteTarget{
			"cognee-ingestion": {Service: "cognee-ingestion", Address: "localhost:9011"},
			"cognee-search":    {Service: "cognee-search", Address: "localhost:9013"},
			"graphiti-ingestion": {Service: "graphiti-ingestion", Address: "localhost:9021"},
			"graphiti-search":    {Service: "graphiti-search", Address: "localhost:9022"},
			"graphiti-store":     {Service: "graphiti-store", Address: "localhost:9024"},
			"memobase-ingestion": {Service: "memobase-ingestion", Address: "localhost:9031"},
			"memobase-context":   {Service: "memobase-context", Address: "localhost:9033"},
			"vnp-event":          {Service: "vnp-event", Address: "localhost:9041"},
			"vnp-search-hub":     {Service: "vnp-search-hub", Address: "localhost:9042"},
			"vnp-admin":          {Service: "vnp-admin", Address: "localhost:9050"},
			"ov-fs":              {Service: "ov-fs", Address: "localhost:9051"},
			"ov-search":          {Service: "ov-search", Address: "localhost:9052"},
			"ov-session":         {Service: "ov-session", Address: "localhost:9053"},
			"ov-resource":        {Service: "ov-resource", Address: "localhost:9054"},
			"zep-user":           {Service: "zep-user", Address: "localhost:9061"},
			"zep-memory":         {Service: "zep-memory", Address: "localhost:9063"},
			"zep-search":         {Service: "zep-search", Address: "localhost:9065"},
			"zep-graph":          {Service: "zep-graph", Address: "localhost:9064"},
			"sm-document":        {Service: "sm-document", Address: "localhost:9071"},
			"sm-memory":          {Service: "sm-memory", Address: "localhost:9072"},
			"sm-search":          {Service: "sm-search", Address: "localhost:9073"},
			"sm-profile":         {Service: "sm-profile", Address: "localhost:9074"},
			"sm-connector":       {Service: "sm-connector", Address: "localhost:9075"},
			"sm-project":         {Service: "sm-project", Address: "localhost:9079"},
		},
		response: []byte(`{"status":"ok","id":"test-123"}`),
	}
}

func (m *mockRegistry) Resolve(service string) (*domain.RouteTarget, error) {
	t, ok := m.services[service]
	if !ok {
		return nil, domain.ErrNotFound.WithMessage("unknown service: " + service)
	}
	return t, nil
}

func (m *mockRegistry) Forward(_ context.Context, _ *domain.RouteTarget, _ []byte) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockRegistry) HealthCheck(service string) (bool, error) {
	_, ok := m.services[service]
	return ok, nil
}

// mockPublisher implements port.EventPublisher for testing.
type mockPublisher struct {
	events []publishedEvent
}

type publishedEvent struct {
	Subject string
	Event   any
}

func (p *mockPublisher) Publish(_ context.Context, subject string, event any) error {
	p.events = append(p.events, publishedEvent{Subject: subject, Event: event})
	return nil
}

// setupTestServer creates a test HTTP server with all handlers wired.
func setupTestServer(t *testing.T) (*httptest.Server, *mockRegistry) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	reg := newMockRegistry()
	pub := &mockPublisher{}

	routeUC := usecase.NewRouteUseCase(reg, pub, logger)

	memoryH := handler.NewMemoryHandler(routeUC, reg, logger)
	cogneeH := handler.NewCogneeHandler(reg, logger)
	graphitiH := handler.NewGraphitiHandler(reg, logger)
	memobaseH := handler.NewMemobaseHandler(reg, logger)
	ovH := handler.NewOpenVikingHandler(reg, logger)
	zepH := handler.NewZepHandler(reg, logger)
	smH := handler.NewSMHandler(reg, logger)
	adminH := handler.NewAdminHandler(reg, logger)

	// Console handlers (SOL-002)
	dashboardH := handler.NewDashboardHandler(reg, logger)
	explorerH := handler.NewExplorerHandler(reg, logger)
	graphH := handler.NewGraphHandler(reg, logger)
	profileH := handler.NewProfileHandler(reg, logger)
	adaptiveH := handler.NewAdaptiveHandler(reg, logger)
	debuggerH := handler.NewDebuggerHandler(reg, logger)
	sessionH := handler.NewSessionHandler(reg, logger)
	governanceH := handler.NewGovernanceHandler(reg, logger)
	pipelineH := handler.NewPipelineHandler(reg, logger)
	infraH := handler.NewInfraHandler(reg, logger)
	observabilityH := handler.NewObservabilityHandler(reg, logger)
	wsH := handler.NewWSHandler(logger)

	router := handler.Router(
		memoryH, cogneeH, graphitiH, memobaseH, ovH, zepH, smH, adminH,
		dashboardH, explorerH, graphH, profileH, adaptiveH,
		debuggerH, sessionH, governanceH, pipelineH, infraH,
		observabilityH, wsH,
		logger,
		nil, // no embedded UI in tests
	)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return server, reg
}

// --- Routing Tests ---

func TestRoute_CogneeSearch(t *testing.T) {
	srv, _ := setupTestServer(t)
	resp, body := doPost(t, srv, "/v1/cognee/search", `{"query":"test"}`)
	assertStatus(t, resp, http.StatusOK)
	assertContains(t, body, "ok")
}

func TestRoute_GraphitiEpisode(t *testing.T) {
	srv, _ := setupTestServer(t)
	resp, body := doPost(t, srv, "/v1/graphiti/episodes", `{"content":"test episode"}`)
	assertStatus(t, resp, http.StatusOK)
	assertContains(t, body, "ok")
}

func TestRoute_MemobaseBlob(t *testing.T) {
	srv, _ := setupTestServer(t)
	resp, body := doPost(t, srv, "/v1/memobase/users/user1/blobs", `{"content":"test"}`)
	assertStatus(t, resp, http.StatusOK)
	assertContains(t, body, "ok")
}

func TestRoute_OVFileRead(t *testing.T) {
	srv, _ := setupTestServer(t)
	resp, _ := doGet(t, srv, "/v1/ov/files/test.txt")
	assertStatus(t, resp, http.StatusOK)
}

func TestRoute_ZepCreateUser(t *testing.T) {
	srv, _ := setupTestServer(t)
	resp, body := doPost(t, srv, "/v1/zep/users", `{"user_id":"u1"}`)
	assertStatus(t, resp, http.StatusOK)
	assertContains(t, body, "ok")
}

func TestRoute_SMSearch(t *testing.T) {
	srv, _ := setupTestServer(t)
	resp, body := doPost(t, srv, "/v1/sm/search", `{"query":"test"}`)
	assertStatus(t, resp, http.StatusOK)
	assertContains(t, body, "ok")
}

func TestRoute_AdminHealth(t *testing.T) {
	srv, _ := setupTestServer(t)
	resp, _ := doGet(t, srv, "/v1/admin/health")
	assertStatus(t, resp, http.StatusOK)
}

func TestRoute_Unknown404(t *testing.T) {
	srv, _ := setupTestServer(t)
	resp, body := doGet(t, srv, "/v1/unknown/path")
	assertStatus(t, resp, http.StatusNotFound)
	assertContains(t, body, "NOT_FOUND")
}

// --- Memory Auto-routing Tests ---

func TestRoute_MemoryStore_Auto(t *testing.T) {
	srv, _ := setupTestServer(t)
	resp, body := doPost(t, srv, "/v1/memory/store", `{"type":"auto","content":"test memory"}`)
	assertStatus(t, resp, http.StatusOK)
	assertContains(t, body, "semantic") // default classifier returns semantic
}

func TestRoute_MemoryRecall(t *testing.T) {
	srv, _ := setupTestServer(t)
	resp, body := doPost(t, srv, "/v1/memory/recall", `{"query":"test recall"}`)
	assertStatus(t, resp, http.StatusOK)
	assertContains(t, body, "ok")
}

func TestRoute_MemoryForget(t *testing.T) {
	srv, _ := setupTestServer(t)
	resp, body := doPost(t, srv, "/v1/memory/forget", `{"target_id":"mem-1"}`)
	assertStatus(t, resp, http.StatusAccepted)
	assertContains(t, body, "accepted")
}

// --- Error Response Tests ---

func TestErrorResponse_JSONFormat(t *testing.T) {
	srv, _ := setupTestServer(t)
	resp, body := doGet(t, srv, "/nonexistent")
	assertStatus(t, resp, http.StatusNotFound)

	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &errResp); err != nil {
		t.Fatalf("failed to parse error JSON: %v", err)
	}
	if errResp.Error.Code != "NOT_FOUND" {
		t.Errorf("error code = %q, want %q", errResp.Error.Code, "NOT_FOUND")
	}
}

func TestCORS_Headers(t *testing.T) {
	srv, _ := setupTestServer(t)
	resp, _ := doOptions(t, srv, "/v1/memory/store")
	assertStatus(t, resp, http.StatusNoContent)

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("CORS origin = %q, want %q", got, "*")
	}
	if got := resp.Header.Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("CORS methods header missing")
	}
}

func TestRequestID_Generated(t *testing.T) {
	srv, _ := setupTestServer(t)
	resp, _ := doGet(t, srv, "/v1/admin/health")
	if got := resp.Header.Get("X-Request-ID"); got == "" {
		t.Error("X-Request-ID header not generated")
	}
}

func TestRequestID_Propagated(t *testing.T) {
	srv, _ := setupTestServer(t)
	req, _ := http.NewRequest("GET", srv.URL+"/v1/admin/health", nil)
	req.Header.Set("X-Request-ID", "custom-id-123")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if got := resp.Header.Get("X-Request-ID"); got != "custom-id-123" {
		t.Errorf("X-Request-ID = %q, want %q", got, "custom-id-123")
	}
}

// --- Helpers ---

func doGet(t *testing.T, srv *httptest.Server, path string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(srv.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, string(body)
}

func doPost(t *testing.T, srv *httptest.Server, path, body string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Post(srv.URL+path, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, string(b)
}

func doOptions(t *testing.T, srv *httptest.Server, path string) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest("OPTIONS", srv.URL+path, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS %s: %v", path, err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return resp, string(b)
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Errorf("status = %d, want %d", resp.StatusCode, want)
	}
}

func assertContains(t *testing.T, body, substr string) {
	t.Helper()
	if !strings.Contains(body, substr) {
		t.Errorf("body %q does not contain %q", body, substr)
	}
}
