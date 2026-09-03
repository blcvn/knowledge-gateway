// Package config provides application configuration via environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for graphiti-store.
type Config struct {
	// Server
	GRPCPort   int    `json:"grpc_port"`
	HealthPort int    `json:"health_port"`
	LogLevel   string `json:"log_level"`

	// Neo4j
	Neo4jURI      string `json:"neo4j_uri"`
	Neo4jUsername  string `json:"neo4j_username"`
	Neo4jPassword string `json:"neo4j_password"`
	Neo4jDatabase string `json:"neo4j_database"`

	// Vector
	VectorDimensions int `json:"vector_dimensions"`

	// NATS
	NATSURL string `json:"nats_url"`

	// OTel
	OTelEndpoint    string `json:"otel_endpoint"`
	OTelServiceName string `json:"otel_service_name"`
}

// Load reads configuration from environment variables with defaults.
func Load() (*Config, error) {
	cfg := &Config{
		GRPCPort:         getEnvInt("GRPC_PORT", 9024),
		HealthPort:       getEnvInt("HEALTH_PORT", 9097),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		Neo4jURI:         getEnv("NEO4J_URI", ""),
		Neo4jUsername:     getEnv("NEO4J_USERNAME", "neo4j"),
		Neo4jPassword:    getEnv("NEO4J_PASSWORD", ""),
		Neo4jDatabase:    getEnv("NEO4J_DATABASE", "neo4j"),
		VectorDimensions: getEnvInt("VECTOR_DIMENSIONS", 1536),
		NATSURL:          getEnv("NATS_URL", "nats://localhost:4222"),
		OTelEndpoint:     getEnv("OTEL_ENDPOINT", "localhost:4317"),
		OTelServiceName:  getEnv("OTEL_SERVICE_NAME", "graphiti-store"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate checks required configuration fields.
func (c *Config) Validate() error {
	if c.Neo4jURI == "" {
		return fmt.Errorf("NEO4J_URI is required")
	}
	if c.Neo4jPassword == "" {
		return fmt.Errorf("NEO4J_PASSWORD is required")
	}
	return nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
