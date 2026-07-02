package observability

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMetricsHandlerReturnsNewCounters(t *testing.T) {
	svc := &Service{
		store: observabilityStoreStub{},
		now:   func() time.Time { return time.Date(2026, 6, 23, 9, 15, 0, 0, time.UTC) },
	}
	handler := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/kg/metrics", nil)
	rec := httptest.NewRecorder()
	handler.Metrics(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, ok := payload["kg_realtime_read_fallback_count"]; !ok {
		t.Fatalf("payload missing kg_realtime_read_fallback_count: %#v", payload)
	}
	if _, ok := payload["kg_graph_scope_conflict_count"]; !ok {
		t.Fatalf("payload missing kg_graph_scope_conflict_count: %#v", payload)
	}
}
