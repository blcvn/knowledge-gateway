package respond

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreatedWritesJSONPayload(t *testing.T) {
	recorder := httptest.NewRecorder()

	Created(recorder, map[string]any{"id": "tenant-1"})

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("content-type = %q", contentType)
	}

	var payload map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if payload["id"] != "tenant-1" {
		t.Fatalf("payload id = %q", payload["id"])
	}
}

func TestErrorWritesEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()

	Error(recorder, StatusFor(CodeValidationFailed), CodeValidationFailed, "Missing field", []map[string]string{
		{"field": "slug", "issue": "required"},
	})

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", recorder.Code)
	}

	var envelope ErrorEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if envelope.Error.Code != CodeValidationFailed {
		t.Fatalf("error code = %q", envelope.Error.Code)
	}
	if envelope.Error.Message != "Missing field" {
		t.Fatalf("error message = %q", envelope.Error.Message)
	}
}

func TestStatusForMappings(t *testing.T) {
	tests := map[string]int{
		CodeBadRequest:             http.StatusBadRequest,
		CodeUnauthorized:           http.StatusUnauthorized,
		CodeForbidden:              http.StatusForbidden,
		CodeNotFound:               http.StatusNotFound,
		CodeValidationFailed:       http.StatusUnprocessableEntity,
		CodeProjectionInconsistent: http.StatusConflict,
		CodeRequestTimedOut:        http.StatusGatewayTimeout,
		CodeInternal:               http.StatusInternalServerError,
	}

	for code, want := range tests {
		if got := StatusFor(code); got != want {
			t.Fatalf("StatusFor(%q) = %d, want %d", code, got, want)
		}
	}
}
