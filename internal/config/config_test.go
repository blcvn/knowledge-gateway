package config

import (
	"os"
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

func TestStringEnvFallsBackOnEmpty(t *testing.T) {
	t.Setenv("KG_EMPTY_VALUE", "   ")
	if got := stringEnv("KG_EMPTY_VALUE", "fallback"); got != "fallback" {
		t.Fatalf("stringEnv() = %q", got)
	}
}

func TestEnvHelpersUseFallbackWhenUnset(t *testing.T) {
	os.Unsetenv("KG_UNKNOWN_INT")
	os.Unsetenv("KG_UNKNOWN_DURATION")

	if got := intEnv("KG_UNKNOWN_INT", 7); got != 7 {
		t.Fatalf("intEnv() = %d", got)
	}
	if got := durationEnv("KG_UNKNOWN_DURATION", time.Minute); got != time.Minute {
		t.Fatalf("durationEnv() = %v", got)
	}
}
