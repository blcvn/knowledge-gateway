// Package config provides unified configuration for the memobase monolith app.
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
	// Default gRPC ports for embedded Memobase services.
	DefaultIngestionPort = 9041
	DefaultEnginePort    = 9042
	DefaultContextPort   = 9043
	DefaultPipelinePort  = 9044
)

// Config is the unified configuration for the memobase monolith app.
type Config struct {
	Server   ServerConfig   `json:"server"`
	Services ServicesConfig `json:"services"`
	Auth     AuthConfig     `json:"auth"`
	Postgres PostgresConfig `json:"postgres"`
	Redis    RedisConfig    `json:"redis"`
	NATS     NATSConfig     `json:"nats"`
	LLM      LLMConfig     `json:"llm"`
	Embedder EmbedderConfig `json:"embedder"`
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

// ServicesConfig holds gRPC ports for embedded memobase services.
type ServicesConfig struct {
	IngestionPort int `json:"ingestion_port"`
	EnginePort    int `json:"engine_port"`
	ContextPort   int `json:"context_port"`
	PipelinePort  int `json:"pipeline_port"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	JWTPublicKey string `json:"jwt_public_key"`
	JWTIssuer    string `json:"jwt_issuer"`
	JWTAudience  string `json:"jwt_audience"`
	DevMode      bool   `json:"dev_mode"`
	RootToken    string `json:"root_token"`
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

// NATSConfig holds NATS JetStream settings for async event communication.
type NATSConfig struct {
	URL        string `json:"url"`
	CredsFile  string `json:"creds_file"`
	StreamName string `json:"stream_name"`
}

// LLMConfig holds LLM provider settings.
type LLMConfig struct {
	Provider    string `json:"provider"`
	APIKey      string `json:"api_key"`
	Model       string `json:"model"`
	SmallModel  string `json:"small_model"`
	BaseURL     string `json:"base_url"`
}

// EmbedderConfig holds embedding provider settings.
type EmbedderConfig struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	APIKey     string `json:"api_key"`
	Dimension  int    `json:"dimension"`
	Enabled    bool   `json:"enabled"`
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
	Context   time.Duration `json:"context"`
	Engine    time.Duration `json:"engine"`
}

// DefaultConfig returns a Config with sensible defaults matching the
// memobase reference spec port assignments and recommended settings.
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
			EnginePort:    DefaultEnginePort,
			ContextPort:   DefaultContextPort,
			PipelinePort:  DefaultPipelinePort,
		},
		Auth: AuthConfig{
			JWTIssuer:   "memobase",
			JWTAudience: "memobase-api",
			DevMode:     false,
		},
		Postgres: PostgresConfig{
			MaxConns: 25,
			MinConns: 5,
		},
		Redis: RedisConfig{
			Addr: "redis:6379",
			DB:   0,
		},
		NATS: NATSConfig{
			URL:        "nats://nats:4222",
			StreamName: "memobase",
		},
		LLM: LLMConfig{
			Provider:   "bifrost",
			Model:      "gpt-4o-mini",
			SmallModel: "gpt-4o-mini",
		},
		Embedder: EmbedderConfig{
			Provider:  "openai",
			Model:     "text-embedding-3-small",
			Dimension: 1536,
			Enabled:   true,
		},
		CORS: CORSConfig{
			AllowedOrigins:   "*",
			AllowCredentials: true,
		},
		Timeout: TimeoutConfig{
			Default:   30 * time.Second,
			Ingestion: 120 * time.Second,
			Context:   30 * time.Second,
			Engine:    300 * time.Second,
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
	if v := os.Getenv("ENGINE_GRPC_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Services.EnginePort = p
		}
	}
	if v := os.Getenv("CONTEXT_GRPC_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Services.ContextPort = p
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
	if v := os.Getenv("ROOT_ACCESS_TOKEN"); v != "" {
		cfg.Auth.RootToken = v
	}

	// Postgres
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.Postgres.DSN = v
	}
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
	if v := os.Getenv("EMBEDDING_ENABLED"); v == "false" || v == "0" {
		cfg.Embedder.Enabled = false
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
	if v := os.Getenv("TIMEOUT_ENGINE"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout.Engine = d
		}
	}

	return cfg
}

// Validate checks that all required configuration fields are set and
// that no port conflicts exist between services.
func (c *Config) Validate() error {
	// Check for port conflicts
	ports := map[int]string{
		c.Services.IngestionPort: "ingestion",
		c.Services.EnginePort:    "engine",
		c.Services.ContextPort:   "context",
		c.Services.PipelinePort:  "pipeline",
		c.Server.RESTPort:        "rest",
		c.Server.MCPPort:         "mcp",
		c.Server.HealthPort:      "health",
	}
	seen := make(map[int]string)
	for port, name := range ports {
		if existing, dup := seen[port]; dup {
			return fmt.Errorf("port %d conflict: %s and %s", port, existing, name)
		}
		seen[port] = name
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
// receive the unified config values.
func (c *Config) SetServiceEnvVars() {
	// Database (used by all memobase services)
	setEnvIfNotEmpty("DATABASE_URL", c.Postgres.DSN)

	// Redis (used by memobase-context for profile caching, gateway for rate limiting)
	setEnvIfNotEmpty("REDIS_ADDR", c.Redis.Addr)
	setEnvIfNotEmpty("REDIS_PASSWORD", c.Redis.Password)

	// NATS (used by ingestion → engine pipeline)
	setEnvIfNotEmpty("NATS_URL", c.NATS.URL)
	setEnvIfNotEmpty("NATS_CREDS_FILE", c.NATS.CredsFile)

	// LLM (used by memobase-engine for profile extraction)
	setEnvIfNotEmpty("LLM_PROVIDER", c.LLM.Provider)
	setEnvIfNotEmpty("LLM_API_KEY", c.LLM.APIKey)
	setEnvIfNotEmpty("LLM_MODEL", c.LLM.Model)
	setEnvIfNotEmpty("LLM_SMALL_MODEL", c.LLM.SmallModel)
	setEnvIfNotEmpty("LLM_BASE_URL", c.LLM.BaseURL)

	// Embedder (used by memobase-event for vector search)
	setEnvIfNotEmpty("EMBEDDER_PROVIDER", c.Embedder.Provider)
	setEnvIfNotEmpty("EMBEDDER_MODEL", c.Embedder.Model)
	setEnvIfNotEmpty("EMBEDDER_API_KEY", c.Embedder.APIKey)

	// Auth (used by gateway)
	setEnvIfNotEmpty("ROOT_ACCESS_TOKEN", c.Auth.RootToken)

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
		"memobase-ingestion": fmt.Sprintf("localhost:%d", c.Services.IngestionPort),
		"memobase-engine":    fmt.Sprintf("localhost:%d", c.Services.EnginePort),
		"memobase-context":   fmt.Sprintf("localhost:%d", c.Services.ContextPort),
		"memobase-pipeline":  fmt.Sprintf("localhost:%d", c.Services.PipelinePort),
	}
}

func setEnvIfNotEmpty(key, value string) {
	if value != "" {
		os.Setenv(key, value)
	}
}
