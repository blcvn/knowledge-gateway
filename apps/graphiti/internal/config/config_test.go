package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Verify service ports
	if cfg.Services.IngestionPort != 9021 {
		t.Errorf("IngestionPort = %d, want 9021", cfg.Services.IngestionPort)
	}
	if cfg.Services.SearchPort != 9022 {
		t.Errorf("SearchPort = %d, want 9022", cfg.Services.SearchPort)
	}
	if cfg.Services.KnowledgePort != 9023 {
		t.Errorf("KnowledgePort = %d, want 9023", cfg.Services.KnowledgePort)
	}
	if cfg.Services.StorePort != 9024 {
		t.Errorf("StorePort = %d, want 9024", cfg.Services.StorePort)
	}
	if cfg.Services.PipelinePort != 9025 {
		t.Errorf("PipelinePort = %d, want 9025", cfg.Services.PipelinePort)
	}

	// Verify server defaults
	if cfg.Server.RESTPort != 8080 {
		t.Errorf("RESTPort = %d, want 8080", cfg.Server.RESTPort)
	}
	if cfg.Server.MCPPort != 8082 {
		t.Errorf("MCPPort = %d, want 8082", cfg.Server.MCPPort)
	}
	if cfg.Server.HealthPort != 9090 {
		t.Errorf("HealthPort = %d, want 9090", cfg.Server.HealthPort)
	}
	if cfg.Server.ShutdownTimeout != 30*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 30s", cfg.Server.ShutdownTimeout)
	}
	if cfg.Server.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.Server.LogLevel, "info")
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Set env vars
	os.Setenv("INGESTION_GRPC_PORT", "19021")
	os.Setenv("NEO4J_URI", "bolt://custom-neo4j:7687")
	os.Setenv("LLM_API_KEY", "test-key")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("AUTH_DEV_MODE", "true")
	defer func() {
		os.Unsetenv("INGESTION_GRPC_PORT")
		os.Unsetenv("NEO4J_URI")
		os.Unsetenv("LLM_API_KEY")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("AUTH_DEV_MODE")
	}()

	cfg := Load()

	if cfg.Services.IngestionPort != 19021 {
		t.Errorf("IngestionPort = %d, want 19021", cfg.Services.IngestionPort)
	}
	if cfg.Neo4j.URI != "bolt://custom-neo4j:7687" {
		t.Errorf("Neo4jURI = %q, want bolt://custom-neo4j:7687", cfg.Neo4j.URI)
	}
	if cfg.LLM.APIKey != "test-key" {
		t.Errorf("LLMAPIKey = %q, want test-key", cfg.LLM.APIKey)
	}
	if cfg.Server.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.Server.LogLevel)
	}
	if !cfg.Auth.DevMode {
		t.Error("DevMode should be true")
	}
}

func TestGatewayServicesMap(t *testing.T) {
	cfg := DefaultConfig()
	m := cfg.GatewayServicesMap()

	expected := map[string]string{
		"graphiti-ingestion": "localhost:9021",
		"graphiti-search":    "localhost:9022",
		"graphiti-knowledge": "localhost:9023",
		"graphiti-store":     "localhost:9024",
		"graphiti-pipeline":  "localhost:9025",
	}

	for k, want := range expected {
		if got := m[k]; got != want {
			t.Errorf("GatewayServicesMap[%q] = %q, want %q", k, got, want)
		}
	}

	if len(m) != len(expected) {
		t.Errorf("GatewayServicesMap has %d entries, want %d", len(m), len(expected))
	}
}

func TestSetServiceEnvVars(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Neo4j.URI = "bolt://test:7687"
	cfg.Neo4j.Username = "testuser"
	cfg.LLM.APIKey = "sk-test-key"
	cfg.Redis.Addr = "redis-test:6379"

	cfg.SetServiceEnvVars()
	defer func() {
		os.Unsetenv("NEO4J_URI")
		os.Unsetenv("NEO4J_USERNAME")
		os.Unsetenv("LLM_API_KEY")
		os.Unsetenv("REDIS_ADDR")
	}()

	if got := os.Getenv("NEO4J_URI"); got != "bolt://test:7687" {
		t.Errorf("NEO4J_URI = %q, want bolt://test:7687", got)
	}
	if got := os.Getenv("NEO4J_USERNAME"); got != "testuser" {
		t.Errorf("NEO4J_USERNAME = %q, want testuser", got)
	}
	if got := os.Getenv("LLM_API_KEY"); got != "sk-test-key" {
		t.Errorf("LLM_API_KEY = %q, want sk-test-key", got)
	}
	if got := os.Getenv("REDIS_ADDR"); got != "redis-test:6379" {
		t.Errorf("REDIS_ADDR = %q, want redis-test:6379", got)
	}
}

func TestValidate(t *testing.T) {
	// DevMode = true should pass even without API key
	cfg := DefaultConfig()
	cfg.Auth.DevMode = true
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() with DevMode=true should pass, got: %v", err)
	}

	// Production mode without API key should fail
	cfg.Auth.DevMode = false
	cfg.LLM.APIKey = ""
	cfg.Auth.JWTPublicKey = "some-key"
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should fail without LLM_API_KEY in production")
	}

	// Production mode without JWT key should fail
	cfg.LLM.APIKey = "sk-test"
	cfg.Auth.JWTPublicKey = ""
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should fail without AUTH_JWT_PUBLIC_KEY in production")
	}
}
