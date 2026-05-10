// Package config provides configuration loading for memobase-pipeline.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the memobase-pipeline service.
type Config struct {
	GRPCPort     int    `json:"grpc_port"`
	HealthPort   int    `json:"health_port"`
	LogLevel     string `json:"log_level"`
	DatabaseURL  string `json:"database_url"`
	RedisURL     string `json:"redis_url"`
	NATSURL      string `json:"nats_url"`
	OTelEndpoint string `json:"otel_endpoint"`
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		GRPCPort:     envInt("GRPC_PORT", 9031),
		HealthPort:   envInt("HEALTH_PORT", 9098),
		LogLevel:     envStr("LOG_LEVEL", "info"),
		DatabaseURL:  envStr("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/vnp_memory?sslmode=disable"),
		RedisURL:     envStr("REDIS_URL", "redis://localhost:6379/0"),
		NATSURL:      envStr("NATS_URL", "nats://localhost:4222"),
		OTelEndpoint: envStr("OTEL_ENDPOINT", ""),
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func envStr(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}
