package write

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"kg-service/internal/httpapi/respond"
)

func TestWriteErrorMapsControlPlaneNotReady(t *testing.T) {
	rec := httptest.NewRecorder()

	writeError(rec, ErrControlPlaneNotReady)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var payload respond.ErrorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Error.Code != respond.CodeServiceUnavailable {
		t.Fatalf("error code = %q, want %q", payload.Error.Code, respond.CodeServiceUnavailable)
	}
}
