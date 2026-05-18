// Package config provides unified configuration for the graphiti monolith app.
//
// It loads configuration from environment variables with sensible defaults,
// then injects them as ENV vars for embedded services that read config via
// os.Getenv() in their own packages.
//
// This bridges the gap between the monolith's unified config and each
// service's individual ENV-based config loading, without modifying any
// service code.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	// Default gRPC ports for embedded Graphiti services.
	DefaultIngestionPort = 9021
	DefaultSearchPort    = 9022
	DefaultKnowledgePort = 9023
	DefaultStorePort     = 9024
	DefaultPipelinePort  = 9025
)

// Config is the unified configuration for the graphiti monolith app.
type Config struct {
	Server   ServerConfig   `json:"server"`
	Services ServicesConfig `json:"services"`
	Auth     AuthConfig     `json:"auth"`
	Postgres PostgresConfig `json:"postgres"`
	Redis    RedisConfig    `json:"redis"`
	NATS     NATSConfig     `json:"nats"`
	Neo4j    Neo4jConfig    `json:"neo4j"`
	LLM      LLMConfig     `json:"llm"`
	Embedder EmbedderConfig `json:"embedder"`
	Reranker RerankerConfig `json:"reranker"`
	CORS     CORSConfig     `json:"cors"`
	Timeout  TimeoutConfig  `json:"timeout"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	RESTPort        int           `json:"rest_port"`
	MCPPort         int           `json:"mcp_port"`
	HealthPort      int           `json:"health_port"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
	LogLevel        string        `json:"log_level"`
}

// ServicesConfig holds gRPC ports for embedded graphiti services.
type ServicesConfig struct {
	IngestionPort int `json:"ingestion_port"`
	SearchPort    int `json:"search_port"`
	KnowledgePort int `json:"knowledge_port"`
	StorePort     int `json:"store_port"`
	PipelinePort  int `json:"pipeline_port"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	JWTPublicKey string `json:"jwt_public_key"`
	JWTIssuer    string `json:"jwt_issuer"`
	JWTAudience  string `json:"jwt_audience"`
	DevMode      bool   `json:"dev_mode"`
}

// PostgresConfig holds PostgreSQL connection settings (for gateway key store).
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

// NATSConfig holds NATS JetStream settings for async event communication.
type NATSConfig struct {
	URL        string `json:"url"`
	CredsFile  string `json:"creds_file"`
	StreamName string `json:"stream_name"`
}

// Neo4jConfig holds Neo4j graph database settings.
type Neo4jConfig struct {
	URI      string `json:"uri"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

// LLMConfig holds LLM provider settings.
type LLMConfig struct {
	Provider    string  `json:"provider"`
	APIKey      string  `json:"api_key"`
	Model       string  `json:"model"`
	SmallModel  string  `json:"small_model"`
	Temperature float64 `json:"temperature"`
	BaseURL     string  `json:"base_url"`
}

// EmbedderConfig holds embedding provider settings.
type EmbedderConfig struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	APIKey     string `json:"api_key"`
	Dimensions int    `json:"dimensions"`
}

// RerankerConfig holds reranker provider settings.
type RerankerConfig struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"api_key"`
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
	Pipeline  time.Duration `json:"pipeline"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			RESTPort:        8080,
			MCPPort:         8082,
			HealthPort:      9090,
			ShutdownTimeout: 30 * time.Second,
			LogLevel:        "info",
		},
		Services: ServicesConfig{
			IngestionPort: DefaultIngestionPort,
			SearchPort:    DefaultSearchPort,
			KnowledgePort: DefaultKnowledgePort,
			StorePort:     DefaultStorePort,
			PipelinePort:  DefaultPipelinePort,
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
			StreamName: "graphiti",
		},
		Neo4j: Neo4jConfig{
			URI:      "bolt://neo4j:7687",
			Username: "neo4j",
			Database: "neo4j",
		},
		LLM: LLMConfig{
			Provider:    "openai",
			Model:       "gpt-4o",
			SmallModel:  "gpt-4o-mini",
			Temperature: 0.0,
		},
		Embedder: EmbedderConfig{
			Provider:   "openai",
			Model:      "text-embedding-3-small",
			Dimensions: 1536,
		},
		Reranker: RerankerConfig{
			Provider: "openai",
			Model:    "gpt-4o-mini",
		},
		CORS: CORSConfig{
			AllowedOrigins:   "*",
			AllowCredentials: true,
		},
		Timeout: TimeoutConfig{
			Default:   30 * time.Second,
			Ingestion: 300 * time.Second,
			Search:    30 * time.Second,
			Pipeline:  600 * time.Second,
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
	if v := os.Getenv("SEARCH_GRPC_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Services.SearchPort = p
		}
	}
	if v := os.Getenv("KNOWLEDGE_GRPC_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Services.KnowledgePort = p
		}
	}
	if v := os.Getenv("STORE_GRPC_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Services.StorePort = p
		}
	}
	if v := os.Getenv("PIPELINE_GRPC_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Services.PipelinePort = p
		}
	}

	// Server
	if v := os.Getenv("REST_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.RESTPort = p
		}
	}
	if v := os.Getenv("MCP_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.MCPPort = p
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
	if v := os.Getenv("NEO4J_DATABASE"); v != "" {
		cfg.Neo4j.Database = v
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
	if v := os.Getenv("LLM_SMALL_MODEL"); v != "" {
		cfg.LLM.SmallModel = v
	}
	if v := os.Getenv("LLM_BASE_URL"); v != "" {
		cfg.LLM.BaseURL = v
	}

	// Embedder
	if v := os.Getenv("EMBEDDER_PROVIDER"); v != "" {
		cfg.Embedder.Provider = v
	}
	if v := os.Getenv("EMBEDDER_MODEL"); v != "" {
		cfg.Embedder.Model = v
	}
	if v := os.Getenv("EMBEDDER_API_KEY"); v != "" {
		cfg.Embedder.APIKey = v
	}

	// Reranker
	if v := os.Getenv("RERANKER_PROVIDER"); v != "" {
		cfg.Reranker.Provider = v
	}
	if v := os.Getenv("RERANKER_MODEL"); v != "" {
		cfg.Reranker.Model = v
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
// receive the unified config values.
func (c *Config) SetServiceEnvVars() {
	// Neo4j (used by graphiti-store)
	setEnvIfNotEmpty("NEO4J_URI", c.Neo4j.URI)
	setEnvIfNotEmpty("NEO4J_USERNAME", c.Neo4j.Username)
	setEnvIfNotEmpty("NEO4J_PASSWORD", c.Neo4j.Password)
	setEnvIfNotEmpty("NEO4J_DATABASE", c.Neo4j.Database)

	// Redis (used by graphiti-search cache)
	setEnvIfNotEmpty("REDIS_ADDR", c.Redis.Addr)
	setEnvIfNotEmpty("REDIS_PASSWORD", c.Redis.Password)

	// NATS (used by ingestion events)
	setEnvIfNotEmpty("NATS_URL", c.NATS.URL)

	// LLM (used by graphiti-knowledge)
	setEnvIfNotEmpty("LLM_PROVIDER", c.LLM.Provider)
	setEnvIfNotEmpty("LLM_API_KEY", c.LLM.APIKey)
	setEnvIfNotEmpty("LLM_MODEL", c.LLM.Model)
	setEnvIfNotEmpty("LLM_SMALL_MODEL", c.LLM.SmallModel)
	setEnvIfNotEmpty("LLM_BASE_URL", c.LLM.BaseURL)

	// Embedder (used by graphiti-knowledge)
	setEnvIfNotEmpty("EMBEDDER_PROVIDER", c.Embedder.Provider)
	setEnvIfNotEmpty("EMBEDDER_MODEL", c.Embedder.Model)
	setEnvIfNotEmpty("EMBEDDER_API_KEY", c.Embedder.APIKey)

	// Reranker
	setEnvIfNotEmpty("RERANKER_PROVIDER", c.Reranker.Provider)
	setEnvIfNotEmpty("RERANKER_MODEL", c.Reranker.Model)

	// Gateway infra
	setEnvIfNotEmpty("POSTGRES_DSN", c.Postgres.DSN)

	// Logging
	setEnvIfNotEmpty("LOG_LEVEL", c.Server.LogLevel)
}

// GatewayServicesMap returns a services→localhost:PORT map for the gateway's
// GRPCRegistry. This overrides the default remote addresses with localhost
// since all services are embedded in the same process.
func (c *Config) GatewayServicesMap() map[string]string {
	return map[string]string{
		"graphiti-ingestion": fmt.Sprintf("localhost:%d", c.Services.IngestionPort),
		"graphiti-search":    fmt.Sprintf("localhost:%d", c.Services.SearchPort),
		"graphiti-knowledge": fmt.Sprintf("localhost:%d", c.Services.KnowledgePort),
		"graphiti-store":     fmt.Sprintf("localhost:%d", c.Services.StorePort),
		"graphiti-pipeline":  fmt.Sprintf("localhost:%d", c.Services.PipelinePort),
	}
}

func setEnvIfNotEmpty(key, value string) {
	if value != "" {
		os.Setenv(key, value)
	}
}
