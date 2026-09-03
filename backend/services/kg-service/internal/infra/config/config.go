// Package config loads configuration for kg-service.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for kg-service.
type Config struct {
	GRPCPort   int    `json:"grpc_port"`
	HealthPort int    `json:"health_port"`
	LogLevel   string `json:"log_level"`

	// Database (pgvector for episodes)
	DatabaseURL string `json:"database_url"`

	// Neo4j (graph store)
	Neo4jURL      string `json:"neo4j_url"`
	Neo4jUser     string `json:"neo4j_user"`
	Neo4jPassword string `json:"neo4j_password"`

	// Embedding service
	EmbeddingURL   string `json:"embedding_url"`
	EmbeddingModel string `json:"embedding_model"`

	// NATS
	NATSURL string `json:"nats_url"`

	// Cognee Python service
	CogneeURL        string `json:"cognee_url"`
	CogneeAPIKey     string `json:"cognee_api_key"`
	CogneeTimeoutSec int    `json:"cognee_timeout_sec"`
	CogneeEnabled    bool   `json:"cognee_enabled"`

	// OTel
	OTelEndpoint string `json:"otel_endpoint"`
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		GRPCPort:         envInt("GRPC_PORT", 9090),
		HealthPort:       envInt("HEALTH_PORT", 9120),
		LogLevel:         envStr("LOG_LEVEL", "info"),
		DatabaseURL:      envStr("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/kg_service?sslmode=disable"),
		Neo4jURL:         envStr("NEO4J_URL", ""),
		Neo4jUser:        envStr("NEO4J_USER", "neo4j"),
		Neo4jPassword:    envStr("NEO4J_PASSWORD", ""),
		EmbeddingURL:     envStr("EMBEDDING_URL", ""),
		EmbeddingModel:   envStr("EMBEDDING_MODEL", "text-embedding-3-small"),
		NATSURL:          envStr("NATS_URL", "nats://localhost:4222"),
		CogneeURL:        envStr("COGNEE_URL", ""),
		CogneeAPIKey:     envStr("COGNEE_API_KEY", ""),
		CogneeTimeoutSec: envInt("COGNEE_TIMEOUT_SECONDS", 120),
		CogneeEnabled:    envBool("COGNEE_ENABLED", false),
		OTelEndpoint:     envStr("OTEL_ENDPOINT", ""),
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
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

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}
