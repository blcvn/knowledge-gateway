package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Service.Name != "cognee-ingestion" {
		t.Errorf("service.name = %q, want %q", cfg.Service.Name, "cognee-ingestion")
	}
	if cfg.GRPC.Port != 9011 {
		t.Errorf("grpc.port = %d, want 9011", cfg.GRPC.Port)
	}
	if cfg.Health.Port != 9091 {
		t.Errorf("health.port = %d, want 9091", cfg.Health.Port)
	}
	if cfg.GRPC.MaxRecvMsgSize != 50*1024*1024 {
		t.Errorf("grpc.max_recv = %d, want 50MB", cfg.GRPC.MaxRecvMsgSize)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	os.Setenv("GRPC_PORT", "9999")
	os.Setenv("LOG_LEVEL", "debug")
	defer os.Unsetenv("GRPC_PORT")
	defer os.Unsetenv("LOG_LEVEL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GRPC.Port != 9999 {
		t.Errorf("grpc.port = %d, want 9999", cfg.GRPC.Port)
	}
	if cfg.Telemetry.LogLevel != "debug" {
		t.Errorf("log_level = %q, want %q", cfg.Telemetry.LogLevel, "debug")
	}
}

func TestValidate_MissingDatabaseURL(t *testing.T) {
	cfg := &Config{
		GRPC:   GRPCConfig{Port: 9011},
		Health: HealthConfig{Port: 9091},
		MinIO:  MinIOConfig{Endpoint: "localhost:9000"},
		NATS:   NATSConfig{URL: "nats://localhost:4222"},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for missing DATABASE_URL")
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	cfg := &Config{
		GRPC:     GRPCConfig{Port: -1},
		Health:   HealthConfig{Port: 9091},
		Postgres: PostgresConfig{URL: "postgres://localhost"},
		MinIO:    MinIOConfig{Endpoint: "localhost:9000"},
		NATS:     NATSConfig{URL: "nats://localhost:4222"},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid port")
	}
}

func TestValidate_AllValid(t *testing.T) {
	cfg := &Config{
		GRPC:     GRPCConfig{Port: 9011},
		Health:   HealthConfig{Port: 9091},
		Postgres: PostgresConfig{URL: "postgres://localhost"},
		MinIO:    MinIOConfig{Endpoint: "localhost:9000"},
		NATS:     NATSConfig{URL: "nats://localhost:4222"},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}
