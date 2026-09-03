package config

import (
	"os"
	"testing"
)

func TestDefaultConfig_HasSensibleDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server.RESTPort != 8080 {
		t.Errorf("expected REST port 8080, got %d", cfg.Server.RESTPort)
	}
	if cfg.Server.HealthPort != 11080 {
		t.Errorf("expected health port 11080, got %d", cfg.Server.HealthPort)
	}
	if cfg.Services.IngestionPort != 9011 {
		t.Errorf("expected ingestion port 9011, got %d", cfg.Services.IngestionPort)
	}
	if cfg.Services.CognifyPort != 9012 {
		t.Errorf("expected cognify port 9012, got %d", cfg.Services.CognifyPort)
	}
	if cfg.Services.SearchPort != 9013 {
		t.Errorf("expected search port 9013, got %d", cfg.Services.SearchPort)
	}
	if cfg.Server.LogLevel != "info" {
		t.Errorf("expected log level 'info', got %q", cfg.Server.LogLevel)
	}
}

func TestLoad_ReadsEnvVars(t *testing.T) {
	// Set test ENV vars
	os.Setenv("REST_PORT", "9999")
	os.Setenv("POSTGRES_DSN", "postgresql://test:pass@localhost/test")
	os.Setenv("NATS_URL", "nats://test:4222")
	os.Setenv("INGESTION_GRPC_PORT", "19011")
	defer func() {
		os.Unsetenv("REST_PORT")
		os.Unsetenv("POSTGRES_DSN")
		os.Unsetenv("NATS_URL")
		os.Unsetenv("INGESTION_GRPC_PORT")
	}()

	cfg := Load()

	if cfg.Server.RESTPort != 9999 {
		t.Errorf("expected REST port 9999 from ENV, got %d", cfg.Server.RESTPort)
	}
	if cfg.Postgres.DSN != "postgresql://test:pass@localhost/test" {
		t.Errorf("expected POSTGRES_DSN from ENV, got %q", cfg.Postgres.DSN)
	}
	if cfg.NATS.URL != "nats://test:4222" {
		t.Errorf("expected NATS_URL from ENV, got %q", cfg.NATS.URL)
	}
	if cfg.Services.IngestionPort != 19011 {
		t.Errorf("expected ingestion port 19011 from ENV, got %d", cfg.Services.IngestionPort)
	}
}

func TestValidate_MissingPostgresDSN(t *testing.T) {
	cfg := DefaultConfig()
	// DSN is empty by default
	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for missing POSTGRES_DSN")
	}
}

func TestValidate_MissingLLMKeyInProd(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Postgres.DSN = "postgresql://test"
	cfg.Auth.DevMode = false
	// LLM.APIKey is empty

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for missing LLM_API_KEY in production")
	}
}

func TestValidate_DevModeSkipsLLMKey(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Postgres.DSN = "postgresql://test"
	cfg.Auth.DevMode = true

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error in dev mode, got: %v", err)
	}
}

func TestValidate_MissingJWTKeyInProd(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Postgres.DSN = "postgresql://test"
	cfg.LLM.APIKey = "sk-test"
	cfg.Auth.DevMode = false
	cfg.Auth.JWTPublicKey = ""

	if err := cfg.Validate(); err == nil {
		t.Error("expected validation error for missing JWT key in production")
	}
}

func TestGatewayServicesMap(t *testing.T) {
	cfg := DefaultConfig()
	m := cfg.GatewayServicesMap()

	expected := map[string]string{
		"cognee-ingestion": "localhost:9011",
		"cognee-cognify":   "localhost:9012",
		"cognee-search":    "localhost:9013",
	}

	for name, addr := range expected {
		if got, ok := m[name]; !ok || got != addr {
			t.Errorf("expected %s=%s, got %s", name, addr, got)
		}
	}
}

func TestSetServiceEnvVars(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Postgres.DSN = "postgresql://test-db"
	cfg.NATS.URL = "nats://test-nats:4222"
	cfg.Neo4j.URI = "bolt://test-neo4j:7687"
	cfg.MinIO.Endpoint = "test-minio:9000"

	cfg.SetServiceEnvVars()

	tests := map[string]string{
		"DATABASE_URL": "postgresql://test-db",
		"POSTGRES_DSN": "postgresql://test-db",
		"NATS_URL":     "nats://test-nats:4222",
		"NEO4J_URI":    "bolt://test-neo4j:7687",
		"MINIO_ENDPOINT": "test-minio:9000",
	}

	for key, want := range tests {
		got := os.Getenv(key)
		if got != want {
			t.Errorf("expected ENV %s=%q, got %q", key, want, got)
		}
	}

	// Cleanup
	for key := range tests {
		os.Unsetenv(key)
	}
}
