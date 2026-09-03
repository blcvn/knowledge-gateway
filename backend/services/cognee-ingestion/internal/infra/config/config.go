// Package config provides structured configuration loading for cognee-ingestion
// via environment variables with sensible production defaults.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the cognee-ingestion service.
type Config struct {
	Service   ServiceConfig   `json:"service"`
	GRPC      GRPCConfig      `json:"grpc"`
	Health    HealthConfig    `json:"health"`
	Postgres  PostgresConfig  `json:"postgres"`
	MinIO     MinIOConfig     `json:"minio"`
	NATS      NATSConfig      `json:"nats"`
	Telemetry TelemetryConfig `json:"telemetry"`
}

// ServiceConfig holds service identity.
type ServiceConfig struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// GRPCConfig holds gRPC server settings.
type GRPCConfig struct {
	Port              int           `json:"port"`
	MaxRecvMsgSize    int           `json:"max_recv_msg_size"`    // bytes
	MaxSendMsgSize    int           `json:"max_send_msg_size"`    // bytes
	ConnectionTimeout time.Duration `json:"connection_timeout"`
}

// HealthConfig holds health check endpoint settings.
type HealthConfig struct {
	Port int `json:"port"`
}

// PostgresConfig holds database connection settings.
type PostgresConfig struct {
	URL          string `json:"url"`
	MaxOpenConns int    `json:"max_open_conns"`
	MaxIdleConns int    `json:"max_idle_conns"`
}

// MinIOConfig holds object storage settings.
type MinIOConfig struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"-"` // sensitive — excluded from JSON
	SecretKey string `json:"-"` // sensitive — excluded from JSON
	Bucket    string `json:"bucket"`
	UseSSL    bool   `json:"use_ssl"`
}

// NATSConfig holds NATS JetStream settings.
type NATSConfig struct {
	URL       string `json:"url"`
	StreamName string `json:"stream_name"`
}

// TelemetryConfig holds observability settings.
type TelemetryConfig struct {
	OTelEndpoint string `json:"otel_endpoint"`
	LogLevel     string `json:"log_level"`
	LogFormat    string `json:"log_format"` // json, text
}

// Load reads configuration from environment variables with production defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Service: ServiceConfig{
			Name:    envStr("SERVICE_NAME", "cognee-ingestion"),
			Version: envStr("SERVICE_VERSION", "0.1.0"),
		},
		GRPC: GRPCConfig{
			Port:              envInt("GRPC_PORT", 9011),
			MaxRecvMsgSize:    envInt("GRPC_MAX_RECV_MSG_SIZE", 50*1024*1024), // 50MB for file uploads
			MaxSendMsgSize:    envInt("GRPC_MAX_SEND_MSG_SIZE", 10*1024*1024), // 10MB
			ConnectionTimeout: time.Duration(envInt("GRPC_CONN_TIMEOUT_SECS", 120)) * time.Second,
		},
		Health: HealthConfig{
			Port: envInt("HEALTH_PORT", 9091),
		},
		Postgres: PostgresConfig{
			URL:          envStr("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/vnp_memory?sslmode=disable"),
			MaxOpenConns: envInt("PG_MAX_OPEN_CONNS", 25),
			MaxIdleConns: envInt("PG_MAX_IDLE_CONNS", 5),
		},
		MinIO: MinIOConfig{
			Endpoint:  envStr("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: envStr("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: envStr("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:    envStr("MINIO_BUCKET", "cognee-ingestion"),
			UseSSL:    envBool("MINIO_USE_SSL", false),
		},
		NATS: NATSConfig{
			URL:        envStr("NATS_URL", "nats://localhost:4222"),
			StreamName: envStr("NATS_STREAM", "cognee"),
		},
		Telemetry: TelemetryConfig{
			OTelEndpoint: envStr("OTEL_ENDPOINT", ""),
			LogLevel:     envStr("LOG_LEVEL", "info"),
			LogFormat:    envStr("LOG_FORMAT", "json"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate checks required configuration fields.
func (c *Config) Validate() error {
	var errs []string

	if c.Postgres.URL == "" {
		errs = append(errs, "DATABASE_URL is required")
	}
	if c.GRPC.Port <= 0 || c.GRPC.Port > 65535 {
		errs = append(errs, fmt.Sprintf("GRPC_PORT must be 1-65535, got %d", c.GRPC.Port))
	}
	if c.Health.Port <= 0 || c.Health.Port > 65535 {
		errs = append(errs, fmt.Sprintf("HEALTH_PORT must be 1-65535, got %d", c.Health.Port))
	}
	if c.MinIO.Endpoint == "" {
		errs = append(errs, "MINIO_ENDPOINT is required")
	}
	if c.NATS.URL == "" {
		errs = append(errs, "NATS_URL is required")
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed: %s", strings.Join(errs, "; "))
	}
	return nil
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

func envBool(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultVal
}
