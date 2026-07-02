package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadUsesEnvironment(t *testing.T) {
	t.Setenv("KG_HTTP_HOST", "127.0.0.1")
	t.Setenv("KG_HTTP_PORT", "9090")
	t.Setenv("KG_POSTGRES_HOST", "db.internal")
	t.Setenv("KG_POSTGRES_PORT", "5433")
	t.Setenv("KG_POSTGRES_USER", "kg")
	t.Setenv("KG_POSTGRES_PASSWORD", "secret")
	t.Setenv("KG_POSTGRES_DATABASE", "kg_service_test")
	t.Setenv("KG_POSTGRES_SSLMODE", "require")
	t.Setenv("KG_POSTGRES_CONN_MAX_LIFETIME", "45m")
	t.Setenv("KG_REDIS_HOST", "cache.internal")
	t.Setenv("KG_REDIS_PORT", "6380")
	t.Setenv("KG_REDIS_DB", "2")
	t.Setenv("VECTOR_ADAPTER", "qdrant")
	t.Setenv("KG_VECTOR_ENDPOINT", "http://qdrant.internal:6333")
	t.Setenv("KG_VECTOR_COLLECTION", "kg_vectors_test")
	t.Setenv("GRAPH_ADAPTER", "neo4j")
	t.Setenv("KG_GRAPH_ENDPOINT", "bolt://neo4j.internal:7687")
	t.Setenv("KG_GRAPH_DATABASE", "neo4j")
	t.Setenv("FTS_ADAPTER", "postgres")
	t.Setenv("KG_RATE_LIMIT_FREE", "100")
	t.Setenv("KG_RATE_LIMIT_PRO", "200")
	t.Setenv("KG_RATE_LIMIT_ENTERPRISE", "300")
	t.Setenv("SYNC_LAG_TOLERANCE_MS", "45000")
	t.Setenv("SYNC_LAG_STUCK_RETRIES", "5")
	t.Setenv("SYNC_ETA_DEFAULT_MS", "7000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTP.Address() != "127.0.0.1:9090" {
		t.Fatalf("HTTP address = %q", cfg.HTTP.Address())
	}
	if cfg.Postgres.DSN() != "postgres://kg:secret@db.internal:5433/kg_service_test?sslmode=require" {
		t.Fatalf("Postgres DSN = %q", cfg.Postgres.DSN())
	}
	if cfg.Postgres.ConnMaxLifetime != 45*time.Minute {
		t.Fatalf("ConnMaxLifetime = %v", cfg.Postgres.ConnMaxLifetime)
	}
	if cfg.Redis.Address() != "cache.internal:6380" {
		t.Fatalf("Redis address = %q", cfg.Redis.Address())
	}
	if cfg.Redis.DB != 2 {
		t.Fatalf("Redis DB = %d", cfg.Redis.DB)
	}
	if cfg.Vector.Kind != "qdrant" {
		t.Fatalf("Vector kind = %q", cfg.Vector.Kind)
	}
	if cfg.Vector.Endpoint != "http://qdrant.internal:6333" {
		t.Fatalf("Vector endpoint = %q", cfg.Vector.Endpoint)
	}
	if cfg.Vector.Collection != "kg_vectors_test" {
		t.Fatalf("Vector collection = %q", cfg.Vector.Collection)
	}
	if cfg.Graph.Kind != "neo4j" {
		t.Fatalf("Graph kind = %q", cfg.Graph.Kind)
	}
	if cfg.Graph.Endpoint != "bolt://neo4j.internal:7687" {
		t.Fatalf("Graph endpoint = %q", cfg.Graph.Endpoint)
	}
	if cfg.Graph.Database != "neo4j" {
		t.Fatalf("Graph database = %q", cfg.Graph.Database)
	}
	if cfg.FTS.Kind != "postgres" {
		t.Fatalf("FTS kind = %q", cfg.FTS.Kind)
	}
	if cfg.RateLimit.FreePerMinute != 100 || cfg.RateLimit.ProPerMinute != 200 || cfg.RateLimit.EnterprisePerMinute != 300 {
		t.Fatalf("RateLimit = %#v", cfg.RateLimit)
	}
	if cfg.SyncLagToleranceMs != 45000 {
		t.Fatalf("SyncLagToleranceMs = %d", cfg.SyncLagToleranceMs)
	}
	if cfg.SyncEtaDefaultMs != 7000 {
		t.Fatalf("SyncEtaDefaultMs = %d", cfg.SyncEtaDefaultMs)
	}
	if cfg.SyncLagStuckRetries != 5 {
		t.Fatalf("SyncLagStuckRetries = %d", cfg.SyncLagStuckRetries)
	}
}

func TestValidateRejectsEmptyHost(t *testing.T) {
	cfg := Config{
		HTTP: HTTPConfig{Host: "", Port: 8082},
		Postgres: PostgresConfig{
			Host:     "127.0.0.1",
			Port:     5432,
			User:     "postgres",
			Database: "kg_service",
		},
		Redis: RedisConfig{Host: "127.0.0.1", Port: 6379},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}
}

func TestValidateRequiresBackendEndpoints(t *testing.T) {
	cfg := Config{
		HTTP: HTTPConfig{Host: "0.0.0.0", Port: 8082},
		Postgres: PostgresConfig{
			Host:     "127.0.0.1",
			Port:     5432,
			User:     "postgres",
			Database: "kg_service",
		},
		Redis:  RedisConfig{Host: "127.0.0.1", Port: 6379},
		Vector: AdapterConfig{Kind: "qdrant"},
		Graph:  AdapterConfig{Kind: "neo4j"},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}
	if got := err.Error(); got == "" {
		t.Fatal("Validate() returned empty error")
	}
}

func TestValidateRequiresNebulaGraphDatabase(t *testing.T) {
	cfg := Config{
		HTTP: HTTPConfig{Host: "0.0.0.0", Port: 8082},
		Postgres: PostgresConfig{
			Host:     "127.0.0.1",
			Port:     5432,
			User:     "postgres",
			Database: "kg_service",
		},
		Redis: RedisConfig{Host: "127.0.0.1", Port: 6379},
		Graph: AdapterConfig{
			Kind:     "nebula",
			Endpoint: "nebula://nebula:9669",
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "KG_GRAPH_DATABASE") {
		t.Fatalf("Validate() error = %q, want KG_GRAPH_DATABASE", err)
	}
}

func TestStringEnvFallsBackOnEmpty(t *testing.T) {
	t.Setenv("KG_EMPTY_VALUE", "   ")
	if got := stringEnv("KG_EMPTY_VALUE", "fallback"); got != "fallback" {
		t.Fatalf("stringEnv() = %q", got)
	}
}

func TestEnvHelpersUseFallbackWhenUnset(t *testing.T) {
	os.Unsetenv("KG_UNKNOWN_INT")
	os.Unsetenv("KG_UNKNOWN_DURATION")

	gotInt, err := intEnv("KG_UNKNOWN_INT", 7)
	if err != nil {
		t.Fatalf("intEnv() error = %v", err)
	}
	if gotInt != 7 {
		t.Fatalf("intEnv() = %d", gotInt)
	}
	gotDuration, err := durationEnv("KG_UNKNOWN_DURATION", time.Minute)
	if err != nil {
		t.Fatalf("durationEnv() error = %v", err)
	}
	if gotDuration != time.Minute {
		t.Fatalf("durationEnv() = %v", gotDuration)
	}
}

func TestEnvHelpersUseFallbackWhenBlank(t *testing.T) {
	t.Setenv("KG_BLANK_INT", "   ")
	t.Setenv("KG_BLANK_DURATION", " ")

	gotInt, err := intEnv("KG_BLANK_INT", 7)
	if err != nil {
		t.Fatalf("intEnv() error = %v", err)
	}
	if gotInt != 7 {
		t.Fatalf("intEnv() = %d, want 7", gotInt)
	}

	gotDuration, err := durationEnv("KG_BLANK_DURATION", time.Minute)
	if err != nil {
		t.Fatalf("durationEnv() error = %v", err)
	}
	if gotDuration != time.Minute {
		t.Fatalf("durationEnv() = %v, want %v", gotDuration, time.Minute)
	}
}

func TestLoadRejectsInvalidIntegerEnv(t *testing.T) {
	t.Setenv("KG_HTTP_PORT", "eightythree")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "KG_HTTP_PORT must be an integer") {
		t.Fatalf("Load() error = %q, want KG_HTTP_PORT parse error", err)
	}
}

func TestLoadRejectsInvalidDurationEnv(t *testing.T) {
	t.Setenv("KG_POSTGRES_CONN_MAX_LIFETIME", "tomorrow")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "KG_POSTGRES_CONN_MAX_LIFETIME must be a duration") {
		t.Fatalf("Load() error = %q, want KG_POSTGRES_CONN_MAX_LIFETIME parse error", err)
	}
}
