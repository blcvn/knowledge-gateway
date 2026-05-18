// Package config provides unit tests for the memobase unified configuration.
package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Services.IngestionPort != 9041 {
		t.Errorf("expected ingestion port 9041, got %d", cfg.Services.IngestionPort)
	}
	if cfg.Services.EnginePort != 9042 {
		t.Errorf("expected engine port 9042, got %d", cfg.Services.EnginePort)
	}
	if cfg.Services.ContextPort != 9043 {
		t.Errorf("expected context port 9043, got %d", cfg.Services.ContextPort)
	}
	if cfg.Services.PipelinePort != 9044 {
		t.Errorf("expected pipeline port 9044, got %d", cfg.Services.PipelinePort)
	}
	if cfg.Server.RESTPort != 8080 {
		t.Errorf("expected REST port 8080, got %d", cfg.Server.RESTPort)
	}
	if cfg.Server.ShutdownTimeout != 30*time.Second {
		t.Errorf("expected shutdown timeout 30s, got %v", cfg.Server.ShutdownTimeout)
	}
	if cfg.NATS.StreamName != "memobase" {
		t.Errorf("expected NATS stream 'memobase', got %q", cfg.NATS.StreamName)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	// Set ENV overrides
	t.Setenv("INGESTION_GRPC_PORT", "19041")
	t.Setenv("ENGINE_GRPC_PORT", "19042")
	t.Setenv("REST_PORT", "18080")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("AUTH_DEV_MODE", "true")
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")
	t.Setenv("NATS_URL", "nats://test:4222")
	t.Setenv("LLM_API_KEY", "sk-test")

	cfg := Load()

	if cfg.Services.IngestionPort != 19041 {
		t.Errorf("expected ingestion port 19041, got %d", cfg.Services.IngestionPort)
	}
	if cfg.Services.EnginePort != 19042 {
		t.Errorf("expected engine port 19042, got %d", cfg.Services.EnginePort)
	}
	if cfg.Server.RESTPort != 18080 {
		t.Errorf("expected REST port 18080, got %d", cfg.Server.RESTPort)
	}
	if cfg.Server.LogLevel != "debug" {
		t.Errorf("expected log level debug, got %q", cfg.Server.LogLevel)
	}
	if !cfg.Auth.DevMode {
		t.Error("expected dev mode true")
	}
	if cfg.Postgres.DSN != "postgres://test:test@localhost/test" {
		t.Errorf("expected test DSN, got %q", cfg.Postgres.DSN)
	}
	if cfg.LLM.APIKey != "sk-test" {
		t.Errorf("expected LLM API key 'sk-test', got %q", cfg.LLM.APIKey)
	}
}

func TestValidatePortConflict(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.DevMode = true
	// Force a port conflict
	cfg.Services.EnginePort = cfg.Services.IngestionPort

	err := cfg.Validate()
	if err == nil {
		t.Error("expected port conflict error")
	}
}

func TestValidateProductionRequirements(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.DevMode = false
	cfg.LLM.APIKey = ""

	err := cfg.Validate()
	if err == nil {
		t.Error("expected LLM_API_KEY required error in production mode")
	}
}

func TestValidateDevModeSkipsRequirements(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Auth.DevMode = true
	cfg.LLM.APIKey = ""

	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected no error in dev mode, got: %v", err)
	}
}

func TestSetServiceEnvVars(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Postgres.DSN = "postgres://memobase@localhost/memobase"
	cfg.Redis.Addr = "localhost:6379"
	cfg.NATS.URL = "nats://localhost:4222"
	cfg.LLM.APIKey = "sk-test-key"
	cfg.LLM.Provider = "openai"

	cfg.SetServiceEnvVars()

	if v := os.Getenv("DATABASE_URL"); v != cfg.Postgres.DSN {
		t.Errorf("expected DATABASE_URL=%q, got %q", cfg.Postgres.DSN, v)
	}
	if v := os.Getenv("NATS_URL"); v != cfg.NATS.URL {
		t.Errorf("expected NATS_URL=%q, got %q", cfg.NATS.URL, v)
	}
	if v := os.Getenv("LLM_API_KEY"); v != cfg.LLM.APIKey {
		t.Errorf("expected LLM_API_KEY=%q, got %q", cfg.LLM.APIKey, v)
	}
}

func TestGatewayServicesMap(t *testing.T) {
	cfg := DefaultConfig()
	m := cfg.GatewayServicesMap()

	expected := map[string]string{
		"memobase-ingestion": "localhost:9041",
		"memobase-engine":    "localhost:9042",
		"memobase-context":   "localhost:9043",
		"memobase-pipeline":  "localhost:9044",
	}

	for k, v := range expected {
		if m[k] != v {
			t.Errorf("expected %s=%q, got %q", k, v, m[k])
		}
	}
}
