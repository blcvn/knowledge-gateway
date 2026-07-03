package observability

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"kg-service/internal/config"
	"kg-service/internal/runtimeobs"
)

func TestLoggerWritesStructuredJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := runtimeobs.NewLoggerWithWriter(config.Config{
		Observability: config.ObservabilityConfig{
			ServiceName:    "kg-service",
			ServiceVersion: "test",
		},
	}, "bootstrap", &buf)

	logger.InfoContext(runtimeobs.WithRequestMeta(httptest.NewRequest("GET", "/", nil).Context(), runtimeobs.NewRequestMeta("req-1", "trace-1", "span-1")), "startup", "addr", "127.0.0.1:8082")

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["service"] != "kg-service" || payload["version"] != "test" {
		t.Fatalf("payload metadata = %#v", payload)
	}
	if payload["component"] != "bootstrap" {
		t.Fatalf("payload component = %#v", payload)
	}
	if payload["request_id"] != "req-1" || payload["trace_id"] != "trace-1" || payload["span_id"] != "span-1" {
		t.Fatalf("payload correlation = %#v", payload)
	}
	if payload["msg"] != "startup" || payload["addr"] != "127.0.0.1:8082" {
		t.Fatalf("payload message = %#v", payload)
	}
}

func TestWorkerLoggerWritesComponentField(t *testing.T) {
	var buf bytes.Buffer
	logger := runtimeobs.NewLoggerWithWriter(config.Config{
		Observability: config.ObservabilityConfig{
			ServiceName:    "kg-service",
			ServiceVersion: "test",
		},
	}, "workers", &buf)

	logger.Printf("projection worker report: processed=%d failed=%d dead_letter=%d", 2, 1, 0)

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["component"] != "workers" {
		t.Fatalf("payload component = %#v", payload)
	}
	if payload["service"] != "kg-service" || payload["version"] != "test" {
		t.Fatalf("payload metadata = %#v", payload)
	}
	if payload["msg"] != "projection worker report: processed=2 failed=1 dead_letter=0" {
		t.Fatalf("payload message = %#v", payload)
	}
}

func TestRequestMetaRoundTripsThroughContext(t *testing.T) {
	ctx := runtimeobs.WithRequestMeta(
		httptest.NewRequest("GET", "/healthz", nil).Context(),
		runtimeobs.NewRequestMeta("req-1", "trace-1", "span-1"),
	)

	meta := runtimeobs.RequestMetaFromContext(ctx)
	if meta.RequestID != "req-1" || meta.TraceID != "trace-1" || meta.SpanID != "span-1" {
		t.Fatalf("meta = %#v", meta)
	}
}
