// Package config provides unified configuration for the cognee monolith app.
// Extends gateway config pattern with cognee-specific settings (Neo4j, Qdrant, LLM, MinIO).
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	// Default gRPC ports for embedded services (matching microservice defaults).
	DefaultIngestionPort = 9011
	DefaultCognifyPort   = 9012
	DefaultSearchPort    = 9013
)

// Config is the unified configuration for the cognee app.
type Config struct {
	Server   ServerConfig   `json:"server"`
	Services ServicesConfig `json:"services"`
	Auth     AuthConfig     `json:"auth"`
	Postgres PostgresConfig `json:"postgres"`
	Redis    RedisConfig    `json:"redis"`
	NATS     NATSConfig     `json:"nats"`
	Neo4j    Neo4jConfig    `json:"neo4j"`
	Qdrant   QdrantConfig   `json:"qdrant"`
	MinIO    MinIOConfig    `json:"minio"`
	LLM      LLMConfig      `json:"llm"`
	CORS     CORSConfig     `json:"cors"`
	Timeout  TimeoutConfig  `json:"timeout"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	RESTPort        int           `json:"rest_port"`
	HealthPort      int           `json:"health_port"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
	LogLevel        string        `json:"log_level"`
}

// ServicesConfig holds gRPC ports for embedded services.
type ServicesConfig struct {
	IngestionPort int `json:"ingestion_port"`
	CognifyPort   int `json:"cognify_port"`
	SearchPort    int `json:"search_port"`
}

// NATSConfig holds NATS JetStream settings for async event communication.
type NATSConfig struct {
	URL        string `json:"url"`
	CredsFile  string `json:"creds_file"`
	StreamName string `json:"stream_name"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	JWTPublicKey string `json:"jwt_public_key"`
	JWTIssuer    string `json:"jwt_issuer"`
	JWTAudience  string `json:"jwt_audience"`
	DevMode      bool   `json:"dev_mode"`
}

// PostgresConfig holds PostgreSQL connection settings.
type PostgresConfig struct {
	DSN      string `json:"dsn"`
	MaxConns int    `json:"max_conns"`
	MinConns int    `json:"min_conns"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// Neo4jConfig holds Neo4j graph database settings.
type Neo4jConfig struct {
	URI      string `json:"uri"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// QdrantConfig holds Qdrant vector store settings.
type QdrantConfig struct {
	URL        string `json:"url"`
	Collection string `json:"collection"`
}

// MinIOConfig holds S3-compatible object storage settings.
type MinIOConfig struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Bucket    string `json:"bucket"`
	UseSSL    bool   `json:"use_ssl"`
}

// LLMConfig holds LLM provider settings.
type LLMConfig struct {
	Provider    string  `json:"provider"`
	APIKey      string  `json:"api_key"`
	Model       string  `json:"model"`
	EmbedModel  string  `json:"embed_model"`
	Temperature float64 `json:"temperature"`
	BaseURL     string  `json:"base_url"`
}

// CORSConfig holds CORS middleware settings.
type CORSConfig struct {
	AllowedOrigins   string `json:"allowed_origins"`
	AllowCredentials bool   `json:"allow_credentials"`
}

// TimeoutConfig holds per-route timeout settings.
type TimeoutConfig struct {
	Default   time.Duration `json:"default"`
	Ingestion time.Duration `json:"ingestion"`
	Search    time.Duration `json:"search"`
	Cognify   time.Duration `json:"cognify"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			RESTPort:        8080,
			HealthPort:      11080,
			ShutdownTimeout: 30 * time.Second,
			LogLevel:        "info",
		},
		Services: ServicesConfig{
			IngestionPort: DefaultIngestionPort,
			CognifyPort:   DefaultCognifyPort,
			SearchPort:    DefaultSearchPort,
		},
		Auth: AuthConfig{
			JWTIssuer:   "vnp-memory",
			JWTAudience: "vnp-api",
			DevMode:     false,
		},
		Postgres: PostgresConfig{
			MaxConns: 20,
			MinConns: 5,
		},
		Redis: RedisConfig{
			Addr: "redis:6379",
			DB:   0,
		},
		NATS: NATSConfig{
			URL:        "nats://nats:4222",
			StreamName: "cognee",
		},
		Neo4j: Neo4jConfig{
			URI:      "bolt://neo4j:7687",
			Username: "neo4j",
		},
		Qdrant: QdrantConfig{
			URL:        "http://qdrant:6333",
			Collection: "cognee",
		},
		MinIO: MinIOConfig{
			Endpoint: "minio:9000",
			Bucket:   "cognee",
			UseSSL:   false,
		},
		LLM: LLMConfig{
			Provider:    "openai",
			Model:       "gpt-4o-mini",
			EmbedModel:  "text-embedding-3-small",
			Temperature: 0.0,
		},
		CORS: CORSConfig{
			AllowedOrigins:   "*",
			AllowCredentials: true,
		},
		Timeout: TimeoutConfig{
			Default:   30 * time.Second,
			Ingestion: 120 * time.Second,
			Search:    10 * time.Second,
			Cognify:   300 * time.Second,
		},
	}
}

// Load reads configuration from environment variables, falling back to defaults.
func Load() *Config {
	cfg := DefaultConfig()

	// Service ports
	if v := os.Getenv("INGESTION_GRPC_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Services.IngestionPort = p
		}
	}
	if v := os.Getenv("COGNIFY_GRPC_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Services.CognifyPort = p
		}
	}
	if v := os.Getenv("SEARCH_GRPC_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Services.SearchPort = p
		}
	}

	// Server
	if v := os.Getenv("REST_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.RESTPort = p
		}
	}
	if v := os.Getenv("HEALTH_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.HealthPort = p
		}
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Server.LogLevel = v
	}
	if v := os.Getenv("SHUTDOWN_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.ShutdownTimeout = d
		}
	}

	// Auth
	if v := os.Getenv("AUTH_JWT_PUBLIC_KEY"); v != "" {
		cfg.Auth.JWTPublicKey = v
	}
	if v := os.Getenv("AUTH_JWT_ISSUER"); v != "" {
		cfg.Auth.JWTIssuer = v
	}
	if v := os.Getenv("AUTH_JWT_AUDIENCE"); v != "" {
		cfg.Auth.JWTAudience = v
	}
	if v := os.Getenv("AUTH_DEV_MODE"); v == "true" || v == "1" {
		cfg.Auth.DevMode = true
	}

	// Postgres
	if v := os.Getenv("POSTGRES_DSN"); v != "" {
		cfg.Postgres.DSN = v
	}
	if v := os.Getenv("POSTGRES_MAX_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Postgres.MaxConns = n
		}
	}

	// Redis
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}

	// NATS
	if v := os.Getenv("NATS_URL"); v != "" {
		cfg.NATS.URL = v
	}
	if v := os.Getenv("NATS_CREDS_FILE"); v != "" {
		cfg.NATS.CredsFile = v
	}
	if v := os.Getenv("NATS_STREAM"); v != "" {
		cfg.NATS.StreamName = v
	}

	// Neo4j
	if v := os.Getenv("NEO4J_URI"); v != "" {
		cfg.Neo4j.URI = v
	}
	if v := os.Getenv("NEO4J_USERNAME"); v != "" {
		cfg.Neo4j.Username = v
	}
	if v := os.Getenv("NEO4J_PASSWORD"); v != "" {
		cfg.Neo4j.Password = v
	}

	// Qdrant
	if v := os.Getenv("QDRANT_URL"); v != "" {
		cfg.Qdrant.URL = v
	}
	if v := os.Getenv("QDRANT_COLLECTION"); v != "" {
		cfg.Qdrant.Collection = v
	}

	// MinIO
	if v := os.Getenv("MINIO_ENDPOINT"); v != "" {
		cfg.MinIO.Endpoint = v
	}
	if v := os.Getenv("MINIO_ACCESS_KEY"); v != "" {
		cfg.MinIO.AccessKey = v
	}
	if v := os.Getenv("MINIO_SECRET_KEY"); v != "" {
		cfg.MinIO.SecretKey = v
	}
	if v := os.Getenv("MINIO_BUCKET"); v != "" {
		cfg.MinIO.Bucket = v
	}

	// LLM
	if v := os.Getenv("LLM_PROVIDER"); v != "" {
		cfg.LLM.Provider = v
	}
	if v := os.Getenv("LLM_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv("LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}
	if v := os.Getenv("LLM_EMBED_MODEL"); v != "" {
		cfg.LLM.EmbedModel = v
	}
	if v := os.Getenv("LLM_BASE_URL"); v != "" {
		cfg.LLM.BaseURL = v
	}

	// Timeouts
	if v := os.Getenv("TIMEOUT_DEFAULT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout.Default = d
		}
	}
	if v := os.Getenv("TIMEOUT_INGESTION"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout.Ingestion = d
		}
	}
	if v := os.Getenv("TIMEOUT_SEARCH"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout.Search = d
		}
	}

	return cfg
}

// Validate checks that all required configuration fields are set.
func (c *Config) Validate() error {
	if c.Postgres.DSN == "" {
		return fmt.Errorf("POSTGRES_DSN is required")
	}
	if c.LLM.APIKey == "" && !c.Auth.DevMode {
		return fmt.Errorf("LLM_API_KEY is required in production mode")
	}
	if !c.Auth.DevMode && c.Auth.JWTPublicKey == "" {
		return fmt.Errorf("AUTH_JWT_PUBLIC_KEY is required in production mode")
	}
	return nil
}

// SetServiceEnvVars injects configuration as environment variables so that
// embedded services (which read config via os.Getenv in their own packages)
// receive the unified config values. This bridges the gap between the
// monolith's config and each service's ENV-based config loading.
func (c *Config) SetServiceEnvVars() {
	// Database
	setEnvIfNotEmpty("DATABASE_URL", c.Postgres.DSN)
	setEnvIfNotEmpty("POSTGRES_DSN", c.Postgres.DSN)

	// NATS
	setEnvIfNotEmpty("NATS_URL", c.NATS.URL)

	// Neo4j
	setEnvIfNotEmpty("NEO4J_URI", c.Neo4j.URI)
	setEnvIfNotEmpty("NEO4J_USERNAME", c.Neo4j.Username)
	setEnvIfNotEmpty("NEO4J_PASSWORD", c.Neo4j.Password)

	// Qdrant
	setEnvIfNotEmpty("QDRANT_URL", c.Qdrant.URL)

	// MinIO
	setEnvIfNotEmpty("MINIO_ENDPOINT", c.MinIO.Endpoint)
	setEnvIfNotEmpty("MINIO_ACCESS_KEY", c.MinIO.AccessKey)
	setEnvIfNotEmpty("MINIO_SECRET_KEY", c.MinIO.SecretKey)
	setEnvIfNotEmpty("MINIO_BUCKET", c.MinIO.Bucket)

	// LLM
	setEnvIfNotEmpty("LLM_GATEWAY_URL", c.LLM.BaseURL)
	setEnvIfNotEmpty("LLM_API_KEY", c.LLM.APIKey)

	// Logging
	setEnvIfNotEmpty("LOG_LEVEL", c.Server.LogLevel)
}

// GatewayServicesMap returns a services→localhost:PORT map for the gateway's
// GRPCRegistry. This overrides the default remote addresses with localhost
// since all services are embedded in the same process.
func (c *Config) GatewayServicesMap() map[string]string {
	return map[string]string{
		"cognee-ingestion": fmt.Sprintf("localhost:%d", c.Services.IngestionPort),
		"cognee-cognify":   fmt.Sprintf("localhost:%d", c.Services.CognifyPort),
		"cognee-search":    fmt.Sprintf("localhost:%d", c.Services.SearchPort),
	}
}

func setEnvIfNotEmpty(key, value string) {
	if value != "" {
		os.Setenv(key, value)
	}
}
