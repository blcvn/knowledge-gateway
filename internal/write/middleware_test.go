package write

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kg-service/internal/access"
)

func TestTraceMiddlewareLogsRequestSummary(t *testing.T) {
	middleware := NewMiddleware()
	var buf bytes.Buffer
	original := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(original)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/kg/write/nodes?graph_version_id=session-1", nil)
	req = req.WithContext(access.ContextWithIdentity(req.Context(), access.Identity{
		TenantID: "tenant-1",
		AppID:    "app-1",
	}))

	rec := httptest.NewRecorder()
	middleware.Trace(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})).ServeHTTP(rec, req)

	logged := buf.String()
	if !strings.Contains(logged, "write request method=POST") {
		t.Fatalf("log = %q, want method", logged)
	}
	if !strings.Contains(logged, "path=/v1/kg/write/nodes?graph_version_id=session-1") {
		t.Fatalf("log = %q, want path", logged)
	}
	if !strings.Contains(logged, "tenant=tenant-1") || !strings.Contains(logged, "app=app-1") {
		t.Fatalf("log = %q, want tenant/app", logged)
	}
	if !strings.Contains(logged, "status=202") {
		t.Fatalf("log = %q, want status", logged)
	}
}
