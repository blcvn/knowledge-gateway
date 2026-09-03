// Package config loads configuration for pipeline-service.
package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for pipeline-service (server + worker shared).
type Config struct {
	// Server
	GRPCPort   int    `json:"grpc_port"`
	HealthPort int    `json:"health_port"`
	LogLevel   string `json:"log_level"`

	// Database
	DatabaseURL string `json:"database_url"`

	// Redis (for worker queue)
	RedisAddr     string `json:"redis_addr"`
	RedisPassword string `json:"redis_password"`
	RedisDB       int    `json:"redis_db"`

	// Worker
	WorkerConcurrency int `json:"worker_concurrency"`
	WorkerQueuePollMs int `json:"worker_queue_poll_ms"`

	// Downstream
	KGServiceAddr string `json:"kg_service_addr"`
	NATSUrl       string `json:"nats_url"`

	// OTel
	OTelEndpoint string `json:"otel_endpoint"`
}

// Load reads config from environment variables.
func Load() *Config {
	return &Config{
		GRPCPort:          envInt("GRPC_PORT", 9090),
		HealthPort:        envInt("HEALTH_PORT", 9160),
		LogLevel:          envStr("LOG_LEVEL", "info"),
		DatabaseURL:       envStr("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/pipeline_service?sslmode=disable"),
		RedisAddr:         envStr("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     envStr("REDIS_PASSWORD", ""),
		RedisDB:           envInt("REDIS_DB", 2),
		WorkerConcurrency: envInt("WORKER_CONCURRENCY", 10),
		WorkerQueuePollMs: envInt("WORKER_QUEUE_POLL_MS", 500),
		KGServiceAddr:     envStr("KG_SERVICE_ADDR", ""),
		NATSUrl:           envStr("NATS_URL", "nats://localhost:4222"),
		OTelEndpoint:      envStr("OTEL_ENDPOINT", ""),
	}
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
