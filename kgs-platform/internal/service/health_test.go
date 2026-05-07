package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

func TestHealthLiveness(t *testing.T) {
	svc := NewHealthService(nil, nil, nil, nil, nil, log.NewStdLogger(io.Discard))
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	svc.Liveness(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("unexpected status payload: %#v", payload)
	}
}

func TestHealthReadinessWithOptionalDeps(t *testing.T) {
	svc := NewHealthService(nil, nil, nil, nil, nil, log.NewStdLogger(io.Discard))
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()

	svc.Readiness(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 with skipped checks, got %d", rr.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if payload["status"] != "ready" {
		t.Fatalf("unexpected status payload: %#v", payload)
	}
}
