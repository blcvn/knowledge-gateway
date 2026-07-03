package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
