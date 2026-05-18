// Package config provides Viper-based configuration loading for vnp-gateway.
package config

import (
	"fmt"
	"time"
)

// Config is the root configuration structure for vnp-gateway.
type Config struct {
	Server    ServerConfig            `mapstructure:"server"`
	Auth      AuthConfig              `mapstructure:"auth"`
	Postgres  PostgresConfig          `mapstructure:"postgres"`
	Redis     RedisConfig             `mapstructure:"redis"`
	RateLimit RateLimitConfig         `mapstructure:"ratelimit"`
	Circuit   CircuitConfig           `mapstructure:"circuit"`
	Timeout   TimeoutConfig           `mapstructure:"timeout"`
	CORS      CORSConfig              `mapstructure:"cors"`
	OTEL      OTELConfig              `mapstructure:"otel"`
	NATS      NATSConfig              `mapstructure:"nats"`
	Services  map[string]string       `mapstructure:"services"` // service-name → grpc-address
}

// ServerConfig holds HTTP/gRPC/MCP server settings.
type ServerConfig struct {
	RESTPort        int           `mapstructure:"REST_PORT"`
	GRPCPort        int           `mapstructure:"GRPC_PORT"`
	MCPPort         int           `mapstructure:"MCP_PORT"`
	HealthPort      int           `mapstructure:"HEALTH_PORT"`
	ShutdownTimeout time.Duration `mapstructure:"SHUTDOWN_TIMEOUT"`
	LogLevel        string        `mapstructure:"LOG_LEVEL"`
	LogFormat       string        `mapstructure:"LOG_FORMAT"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	JWTPublicKey string `mapstructure:"AUTH_JWT_PUBLIC_KEY"`
	JWTIssuer    string `mapstructure:"AUTH_JWT_ISSUER"`
	JWTAudience  string `mapstructure:"AUTH_JWT_AUDIENCE"`
	APIKeyPrefix string `mapstructure:"AUTH_API_KEY_PREFIX"`
	DevMode      bool   `mapstructure:"AUTH_DEV_MODE"`
}

// PostgresConfig holds PostgreSQL connection settings.
type PostgresConfig struct {
	DSN      string `mapstructure:"POSTGRES_DSN"`
	MaxConns int    `mapstructure:"POSTGRES_MAX_CONNS"`
	MinConns int    `mapstructure:"POSTGRES_MIN_CONNS"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr     string `mapstructure:"REDIS_ADDR"`
	Password string `mapstructure:"REDIS_PASSWORD"`
	DB       int    `mapstructure:"REDIS_DB"`
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	Enabled       bool `mapstructure:"RATELIMIT_ENABLED"`
	FreeRPM       int  `mapstructure:"RATELIMIT_FREE_RPM"`
	ProRPM        int  `mapstructure:"RATELIMIT_PRO_RPM"`
	EnterpriseRPM int  `mapstructure:"RATELIMIT_ENTERPRISE_RPM"`
}

// CircuitConfig holds circuit breaker settings.
type CircuitConfig struct {
	MaxFailures int           `mapstructure:"CB_MAX_FAILURES"`
	Timeout     time.Duration `mapstructure:"CB_TIMEOUT"`
	MaxRequests int           `mapstructure:"CB_MAX_REQUESTS"`
}

// TimeoutConfig holds per-route timeout settings.
type TimeoutConfig struct {
	Default   time.Duration `mapstructure:"TIMEOUT_DEFAULT"`
	Ingestion time.Duration `mapstructure:"TIMEOUT_INGESTION"`
	Search    time.Duration `mapstructure:"TIMEOUT_SEARCH"`
	MCP       time.Duration `mapstructure:"TIMEOUT_MCP"`
}

// CORSConfig holds CORS settings.
type CORSConfig struct {
	AllowedOrigins   string `mapstructure:"CORS_ALLOWED_ORIGINS"`
	AllowCredentials bool   `mapstructure:"CORS_ALLOW_CREDENTIALS"`
	MaxAge           int    `mapstructure:"CORS_MAX_AGE"`
}

// OTELConfig holds OpenTelemetry settings.
type OTELConfig struct {
	Endpoint    string `mapstructure:"OTEL_ENDPOINT"`
	ServiceName string `mapstructure:"OTEL_SERVICE_NAME"`
	Enabled     bool   `mapstructure:"METRICS_ENABLED"`
}

// NATSConfig holds NATS connection settings.
type NATSConfig struct {
	URL       string `mapstructure:"NATS_URL"`
	CredsFile string `mapstructure:"NATS_CREDS_FILE"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			RESTPort:        8080,
			GRPCPort:        8081,
			MCPPort:         8082,
			HealthPort:      11080,
			ShutdownTimeout: 30 * time.Second,
			LogLevel:        "info",
			LogFormat:       "json",
		},
		Auth: AuthConfig{
			JWTIssuer:    "vnp-memory",
			JWTAudience:  "vnp-api",
			APIKeyPrefix: "vnp_",
			DevMode:      false,
		},
		Postgres: PostgresConfig{
			MaxConns: 20,
			MinConns: 5,
		},
		Redis: RedisConfig{
			Addr: "redis:6379",
			DB:   0,
		},
		RateLimit: RateLimitConfig{
			Enabled:       true,
			FreeRPM:       60,
			ProRPM:        600,
			EnterpriseRPM: 6000,
		},
		Circuit: CircuitConfig{
			MaxFailures: 5,
			Timeout:     60 * time.Second,
			MaxRequests: 3,
		},
		Timeout: TimeoutConfig{
			Default:   30 * time.Second,
			Ingestion: 120 * time.Second,
			Search:    10 * time.Second,
			MCP:       300 * time.Second,
		},
		CORS: CORSConfig{
			AllowedOrigins:   "*",
			AllowCredentials: true,
			MaxAge:           86400,
		},
		OTEL: OTELConfig{
			Endpoint:    "otel-collector:4317",
			ServiceName: "vnp-gateway",
			Enabled:     true,
		},
		NATS: NATSConfig{
			URL: "nats://nats:4222",
		},
		Services: defaultServiceAddresses(),
	}
}

// defaultServiceAddresses returns the standard gRPC addresses for all 35 services.
func defaultServiceAddresses() map[string]string {
	return map[string]string{
		"cognee-ingestion":  "cognee-ingestion:9011",
		"cognee-cognify":    "cognee-cognify:9012",
		"cognee-search":     "cognee-search:9013",
		"graphiti-ingestion": "graphiti-ingestion:9021",
		"graphiti-search":    "graphiti-search:9022",
		"graphiti-knowledge": "graphiti-knowledge:9023",
		"graphiti-store":     "graphiti-store:9024",
		"memobase-ingestion": "memobase-ingestion:9031",
		"memobase-engine":    "memobase-engine:9032",
		"memobase-context":   "memobase-context:9033",
		"vnp-event":          "vnp-event:9041",
		"vnp-search-hub":     "vnp-search-hub:9042",
		"vnp-admin":          "vnp-admin:9050",
		"ov-fs":              "ov-fs:9051",
		"ov-search":          "ov-search:9052",
		"ov-session":         "ov-session:9053",
		"ov-resource":        "ov-resource:9054",
		"ov-crypto":          "ov-crypto:9055",
		"ov-admin":           "ov-admin:9056",
		"zep-user":           "zep-user:9061",
		"zep-thread":         "zep-thread:9062",
		"zep-memory":         "zep-memory:9063",
		"zep-graph":          "zep-graph:9064",
		"zep-search":         "zep-search:9065",
		"zep-admin":          "zep-admin:9066",
		"sm-document":        "sm-document:9071",
		"sm-memory":          "sm-memory:9072",
		"sm-search":          "sm-search:9073",
		"sm-profile":         "sm-profile:9074",
		"sm-connector":       "sm-connector:9075",
		"sm-mcp":             "sm-mcp:9076",
		"sm-auth":            "sm-auth:9077",
		"sm-analytics":       "sm-analytics:9078",
		"sm-project":         "sm-project:9079",
		// Console-specific services (SOL-002 / T14)
		"vnp-platform":       "vnp-platform:9043",
		"sm-engine":          "sm-engine:9080",
		"zep-core":           "zep-core:9067",
	}
}

// Validate checks required configuration values.
func (c *Config) Validate() error {
	if c.Server.RESTPort == 0 {
		return fmt.Errorf("REST_PORT is required")
	}
	if !c.Auth.DevMode && c.Auth.JWTPublicKey == "" {
		return fmt.Errorf("AUTH_JWT_PUBLIC_KEY is required when AUTH_DEV_MODE=false")
	}
	return nil
}

// Load reads configuration from environment variables with defaults.
func Load() *Config {
	cfg := DefaultConfig()

	// In production, overlay with Viper env binding:
	// viper.AutomaticEnv()
	// viper.Unmarshal(cfg)
	// For now, return defaults
	return cfg
}
